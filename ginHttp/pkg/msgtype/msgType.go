package msgtype

type Block struct {
	Number    uint64       `json:"number"`
	BlockID   string       `json:"blockHash"`
	Leader    string       `json:"leaderID"`
	PrevBlock string       `json:"prevBlockHash"`
	SubNet    string       `json:"subNet"`
	TxCount   int          `json:"txCount"`
	TxList    []string     `json:"txList"`
	TimeStamp string       `json:"timeStamp"`
	Receipt   BlockReceipt `json:"BlockReceipt"`
}

type BlockReceipt struct {
	ReceiptID string               `json:"receiptID"`
	TxReceipt []TransactionReceipt `json:"txReceipt"`
	WriteSet  []WriteEle           `json:"WriteSet"`
}

type TransactionReceipt struct {
	Valid  byte     `json:"valid"`
	Result []string `json:"result"`
}

type WriteEle struct {
	Value string `json:"value"`
}

type Transaction struct {
	TxID      string             `json:"txID"`
	Sender    string             `json:"sender"`
	Version   string             `json:"version"`
	Signature []byte             `json:"signature"`
	Contract  string             `json:"contract"`
	Function  string             `json:"function"`
	Args      []string           `json:"args"`
	Receipt   TransactionReceipt `json:"receipt"`
}

type DPNetWork struct {
	Booters []string       `json:"booters"`
	Leaders []string       `json:"leaders"`
	SubNets []DPSubNetwork `json:"subnets"`
}

type DPSubNetwork struct {
	Leader    string   `json:"leader"`
	NetID     string   `json:"netID"`
	Followers []string `json:"followers"`
}

type DPNode struct {
	NodeID string `json:"nodeID"`
	Role   string `json:"role"`
	NetID  string `json:"netID"`

	State bool `json:"state"`

	LeaderID string `json:"leaderID"`
}

type address string
type tokenId string
type url string
type transTime int
type credit int
type Credit map[address]credit
type Asset struct {
	NftName           string                            `json:"nftName"`               // the full name of NFT
	NftSymbol         string                            `json:"nftSymbol"`             // the abbr/symbol of NFT
	Owner             string                            `json:"owner"`                 // creator of the NFT ledger
	AssetOf           map[address]map[tokenId]url       `json:"assetOf"`               // collection of NFT, NFT metadata, and NFT holders
	AssetHistory      map[tokenId]map[transTime]address `json:"assetHistory"`          // transaction records of NFT, address record the current owner
	AssetApprove      map[tokenId]address               `json:"assetApprove"`          // authorized account of NFT
	AssetExist        map[url]bool                      `json:"assetExist"`            // check the existence of a NFT through the metadata
	AssetTokenIdExist map[tokenId]bool                  `json:"assetTokenIdExist"`     // check the existence of a NFT through the NFT id
	TotalSupply       int                               `json:"totalSupply,omitempty"` // total assets of a NFT. 0 means unlimited NFT and non-0 means limited edition NFT
}
type StampList struct {
	StampName tokenId `json:"stampName"`
	Link      url     `json:"hashValue"`
	MetaData  string  `json:"sourceFile"`
}
