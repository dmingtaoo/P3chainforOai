package accounts

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"fmt"
	"os"
	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	loglogrus "p3Chain/log_logrus"
	"sync"
	"time"
)

var (
	ErrLocked = errors.New("account is locked")
	ErrNoKeys = errors.New("no keys in store")
)

type Manager struct {
	keyStore keys.KeyStore
	unlocked map[common.Address]*unlocked
	mutex    sync.RWMutex
}

type unlocked struct {
	*keys.Key
	abort chan struct{}
}

func NewManager(keyStore keys.KeyStore) *Manager {
	return &Manager{
		keyStore: keyStore,
		unlocked: make(map[common.Address]*unlocked),
	}
}

func (am *Manager) HasAccount(addr common.Address) bool {
	accounts, _ := am.Accounts()
	for _, acct := range accounts {
		if acct.Address == addr {
			return true
		}
	}
	return false
}

func (am *Manager) DeleteAccount(address common.Address, auth string) error {
	return am.keyStore.DeleteKey(address, auth)
}

func (am *Manager) Sign(a Account, toSign []byte) (signature []byte, err error) {
	am.mutex.RLock()
	unlockedKey, found := am.unlocked[a.Address]
	am.mutex.RUnlock()
	if !found {
		return nil, ErrLocked
	}
	signature, err = crypto.Sign(toSign, unlockedKey.PrivateKey)
	return signature, err
}

func (am *Manager) SignHash(a Account, hash common.Hash) (signature []byte, err error) {
	am.mutex.RLock()
	unlockedKey, found := am.unlocked[a.Address]
	am.mutex.RUnlock()
	if !found {
		return nil, ErrLocked
	}
	signature, err = crypto.SignHash(hash, unlockedKey.PrivateKey)
	return signature, err
}

// unlock indefinitely
func (am *Manager) Unlock(addr common.Address, keyAuth string) error {
	return am.TimedUnlock(addr, keyAuth, 0)
}

// Unlock unlocks the account with the given address. The account
// stays unlocked for the duration of timeout
// it timeout is 0 the account is unlocked for the entire session
func (am *Manager) TimedUnlock(addr common.Address, keyAuth string, timeout time.Duration) error {
	key, err := am.keyStore.GetKey(addr, keyAuth)
	if err != nil {
		loglogrus.Log.Warnf("Account Manager: Unlock address is faile, becaues can't get key -- %v\n", err)
		return err
	}
	var u *unlocked
	am.mutex.Lock()
	defer am.mutex.Unlock()
	var found bool
	u, found = am.unlocked[addr]
	if found {
		// terminate dropLater for this key to avoid unexpected drops.
		if u.abort != nil {
			close(u.abort)
		}
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go am.expire(addr, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	am.unlocked[addr] = u
	return nil
}

func (am *Manager) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		am.mutex.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if am.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(am.unlocked, addr)
		}
		am.mutex.Unlock()
	}
}

func (am *Manager) NewAccount(auth string) (Account, error) {
	key, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		loglogrus.Log.Warnf("Account Manager: Can't Generate New Key,err: %v\n", err)
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
}

func (am *Manager) AddressByIndex(index int) (addr string, err error) {
	var addrs []common.Address
	addrs, err = am.keyStore.GetKeyAddresses()
	if err != nil {
		return
	}
	if index < 0 || index >= len(addrs) {
		err = fmt.Errorf("index out of range: %d (should be 0-%d)", index, len(addrs)-1)
	} else {
		addr = addrs[index].Hex()
	}
	return
}

func (am *Manager) Accounts() ([]Account, error) {
	addresses, err := am.keyStore.GetKeyAddresses()
	if os.IsNotExist(err) {
		loglogrus.Log.Warnf("Account Manager: Current Node could not find any key file from store!\n")
		return nil, ErrNoKeys
	} else if err != nil {
		loglogrus.Log.Warnf("Account Manager: Current Node could not get addresss, err: %v\n", err)
		return nil, err
	}
	accounts := make([]Account, len(addresses))
	for i, addr := range addresses {
		accounts[i] = Account{
			Address: addr,
		}
	}
	return accounts, nil
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

// USE WITH CAUTION = this will save an unencrypted private key on disk
// no cli or js interface
func (am *Manager) Export(path string, addr common.Address, keyAuth string) error {
	key, err := am.keyStore.GetKey(addr, keyAuth)
	if err != nil {
		return err
	}
	return crypto.SaveECDSA(path, key.PrivateKey)
}

func (am *Manager) Import(path string, keyAuth string) (Account, error) {
	privateKeyECDSA, err := crypto.LoadECDSA(path)
	if err != nil {
		return Account{}, err
	}
	key := keys.NewKeyFromECDSA(privateKeyECDSA)
	if err = am.keyStore.StoreKey(key, keyAuth); err != nil {
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
}

func (am *Manager) Update(addr common.Address, authFrom, authTo string) (err error) {
	var key *keys.Key
	key, err = am.keyStore.GetKey(addr, authFrom)

	if err == nil {
		err = am.keyStore.StoreKey(key, authTo)
		if err == nil {
			am.keyStore.Cleanup(addr)
		}
	}
	return
}

func (am *Manager) ImportPreSaleKey(keyJSON []byte, password string) (acc Account, err error) {
	var key *keys.Key
	key, err = keys.ImportPreSaleKey(am.keyStore, keyJSON, password)
	if err != nil {
		return
	}
	return Account{Address: key.Address}, nil
}

func (am *Manager) GetAccount(addr common.Address) (Account, error) {
	accounts, _ := am.Accounts()
	for _, acct := range accounts {
		if acct.Address == addr {
			return acct, nil
		}
	}
	loglogrus.Log.Warnf("Account Manager: The account (%x) can't be found!\n", addr)
	return Account{}, fmt.Errorf("no such account is found")
}
func(am *Manager) BackKeystore()keys.KeyStore{
	return am.keyStore
}