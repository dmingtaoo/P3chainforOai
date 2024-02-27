package cli

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"p3Chain/p2p/server"
	"p3Chain/p2p/whisper"
	"p3Chain/utils"
	"strconv"
	"strings"
	"time"
)

func (c *CommandLine) doAction(line string, wps *server.Server, shh *whisper.Whisper) error {
	parameters := strings.Split(strings.TrimSpace(line), "|")
	if len(parameters) == 0 {
		printActionUsage()
		return errors.New("error in doAction: null line is parsed")
	}
	switch parameters[0] {
	case "SEND":
		WPS_LOGGER.Infof("action command: SEND\n")
		err := c.doSEND(parameters[1:], shh)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "WATCH":
		WPS_LOGGER.Infof("action command: WATCH\n")
		err := c.doWATCH(parameters[1:], shh)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "SAVESELFURL":
		WPS_LOGGER.Infof("action command: SAVESELFURL\n")
		err := c.doSAVESELFURL(wps)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "SELFKEYREGISTER":
		WPS_LOGGER.Infof("action command: SELFKEYREGISTER\n")
		err := c.doSELFKEYREGISTER(wps, shh)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "REGISTERPUBKEY":
		WPS_LOGGER.Infof("action command: REGISTERPUBKEY\n")
		err := c.doREGISTERPUBKEY()
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "NEWPRVKEY":
		WPS_LOGGER.Infof("action command: NEWPRVKEY\n")
		err := c.doNEWPRVKEY(parameters[1:], shh)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "SLEEP":
		WPS_LOGGER.Infof("action command: SLEEP\n")
		err := c.doSLEEP(parameters[1:])
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	case "SAVESELFPUBKEY":
		WPS_LOGGER.Infof("action command: SAVESELFPUBKEY\n")
		err := c.doSAVESELFPUBKEY(wps)
		if err != nil {
			WPS_LOGGER.Infof("error: %v.\n", err)
		}
	default:
		printActionUsage()
	}

	return nil
}

func printActionUsage() {
	fmt.Println("Input action to make whisper client (or say server) work.")
	fmt.Println("Input Q, q, quit, exit, or others to quit this action interpreter.")
	fmt.Println("You can also use auto mode to read and do actions from actionlist.")
	fmt.Println("ACTION USAGE:")
	fmt.Println("SEND|FROM|TO|TTL|MESSAGE")
	fmt.Println("SENDFILE|FROM|TO|TTL") // TODO: in the further future
	fmt.Println("WATCH|TO|FUNC")
	fmt.Println("SAVESELFURL")
	fmt.Println("SELFKEYREGISTER")
	fmt.Println("REGISTERPUBKEY")
	fmt.Println("NEWPRVKEY|NAME")
	fmt.Println("SLEEP|SECONDS")
	fmt.Println("SAVESELFPUBKEY")
}

func (c *CommandLine) doSAVESELFURL(wps *server.Server) error {
	url := wps.Self().String()
	err := utils.SaveSelfUrl(url, SELF_FILE)
	return err
}

func (c *CommandLine) doSELFKEYREGISTER(wps *server.Server, shh *whisper.Whisper) error {
	shh.InjectIdentity(wps.PrivateKey)
	c.registerPrvKey["SELF"] = *wps.PrivateKey
	c.registerPubKey["SELF"] = wps.PrivateKey.PublicKey
	return nil
}

func (c *CommandLine) doSEND(parameters []string, shh *whisper.Whisper) error {
	if len(parameters) != 4 {
		return errors.New("error in action send: incorrect parameters")
	}
	var from ecdsa.PrivateKey
	var to ecdsa.PublicKey
	var ttl time.Duration
	var ok bool
	from, ok = c.registerPrvKey[parameters[0]]
	if !ok {
		return errors.New("error in action send: from not right")
	}
	to, ok = c.registerPubKey[parameters[1]]
	if !ok {
		return errors.New("error in action send: to not right")
	}

	if parameters[2] == "D" || parameters[2] == "Default" {
		ttl = whisper.DefaultTTL
	} else {
		sec, err := strconv.Atoi(parameters[2])
		if err != nil {
			return err
		}
		ttl = time.Duration(sec) * time.Second
	}

	msg := whisper.NewMessage([]byte(parameters[3]))
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{
		From: &from,
		To:   &to,
		TTL:  ttl,
	})
	if err != nil {
		return err
	}
	err = shh.Send(envelope)
	if err != nil {
		return err
	}
	WPS_LOGGER.Infof("Message send: From:%v,To:%v,Content:%s\n", from, to, parameters[3])
	return nil
}

func (c *CommandLine) doWATCH(parameters []string, shh *whisper.Whisper) error {
	if len(parameters) != 2 {
		return errors.New("error in action watch: incorrect parameters")
	}
	var to ecdsa.PublicKey
	var fn func(msg *whisper.Message)
	var ok bool
	to, ok = c.registerPubKey[parameters[0]]
	if !ok {
		return errors.New("error in action watch: to not right")
	}
	if parameters[1] == "D" || parameters[1] == "Default" {
		fn = func(msg *whisper.Message) {
			fmt.Printf("Message received: %s, signed with 0x%x.\n", string(msg.Payload), msg.Signature)
			WPS_LOGGER.Infof("Message received: To:%v,Time:%v,Content:%s\n", *msg.To, time.Since(msg.Sent), string(msg.Payload))
		}
	} else {
		return errors.New("error in action watch: no such function")
	}
	shh.Watch(whisper.Filter{
		To: &to,
		Fn: fn,
	})
	return nil
}

func (c *CommandLine) doREGISTERPUBKEY() error {
	names, pbks, err := utils.LoadPublicKeyDir(PUBKEY_DIR)
	if err != nil {
		return nil
	}
	for i := 0; i < len(names); i++ {
		c.registerPubKey[names[i]] = pbks[i]
	}
	return nil
}

func (c *CommandLine) doNEWPRVKEY(parameters []string, shh *whisper.Whisper) error {
	if len(parameters) != 1 {
		return errors.New("error in action newprvkey: incorrect parameters")
	}
	id := shh.NewIdentity()
	c.registerPrvKey[parameters[0]] = *id
	return nil
}

func (c *CommandLine) doSLEEP(parameters []string) error {
	if len(parameters) != 1 {
		return errors.New("error in action sleep: incorrect parameters")
	}

	sec, err := strconv.Atoi(parameters[0])
	if err != nil {
		return err
	}
	interval := time.Duration(sec) * time.Second
	time.Sleep(interval)
	return nil
}

func (c *CommandLine) doSAVESELFPUBKEY(wps *server.Server) error {
	err := utils.SaveSelfPublicKey(wps.PrivateKey.PublicKey, SELFPUBKEY_FILE)
	return err
}
