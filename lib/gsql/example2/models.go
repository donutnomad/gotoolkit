package example2

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
type NFTStatus int
type RORRequestStatus int

type RORRequest struct {
	Model

	NFTContract string    `gorm:"not null;column:nft_contract;size:100;default:''"` // NFT合约地址
	NFTID       int64     `gorm:"not null;column:nft_id;default:0;index"`           // NFT唯一ID
	NFTStatus   NFTStatus `gorm:"not null;column:nft_status;default:0"`             // NFT的状态
	Updated     bool      `gorm:"not null;column:updated;default:false"`            // 是否需要更新

	TokenID       uint   `gorm:"not null;column:token_id;default:0"`              // Token ID
	TokenName     string `gorm:"not null;column:token_name;size:100;default:''"`  // Token名称
	TokenSymbol   string `gorm:"not null;column:token_symbol;size:50;default:''"` // Token符号
	TokenDecimals uint8  `gorm:"not null;column:token_decimals;default:0"`        // Token小数位

	Creator  string          `gorm:"not null;column:creator;size:100;default:''"`          // 创建者地址
	From     string          `gorm:"not null;column:from;size:100;default:''"`             // 发送地址
	Amount   decimal.Decimal `gorm:"not null;column:amount;type:decimal(40,18);default:0"` // 金额
	Receiver string          `gorm:"not null;column:receiver;size:100;default:''"`         // 接收地址
	PartyA   string          `gorm:"not null;column:party_a;size:100;default:''"`          // party_a地址

	ExecutionDateStartTime int64 `gorm:"not null;column:execution_date_start_time;default:0"` // 执行开始时间
	ExecutionDateEndTime   int64 `gorm:"not null;column:execution_date_end_time;default:0"`   // 执行结束时间
	ExecutionDateDay       int32 `gorm:"not null;column:execution_date_day;default:0"`        // 执行天数
	ExecutionDateType      uint8 `gorm:"not null;column:execution_date_type;default:0"`       // 日期类型1,2,3,4, 仅作为标记位

	LogicAnd bool `gorm:"not null;column:logic_and;default:false"` // 是否逻辑与

	Conditions datatypes.JSONSlice[string] `gorm:"not null;column:conditions;type:json"` // 条件

	RecordCreatedAt int64 `gorm:"not null;column:record_created_at;default:0"` // 创建记录时间
	NFTCreatedAt    int64 `gorm:"not null;column:nft_created_at;default:0"`    // NFT被创建的时间

	Status RORRequestStatus `gorm:"not null;column:status;default:0"` // 状态

	UpdateAtBlockNumber    uint64 `gorm:"not null;column:update_at_block_number;default:0"`    // NFT状态同步的区块
	UpdateAtBlockTimestamp int64  `gorm:"not null;column:update_at_block_timestamp;default:0"` // NFT状态同步的区块时间
}

func (RORRequest) TableName() string {
	return "abt_ror_request"
}

// RORTransferBalance 因为只有第一任拥有者才能转账，所以说，只要有人转给他，都算是一个记录，且这个记录只有加，没有减
type RORTransferBalance struct {
	Model
	NFTContract string                      `gorm:"not null;column:nft_contract;size:100;default:'';index"` // NFT合约地址
	NFTID       int64                       `gorm:"not null;column:nft_id;default:0;index"`
	Account     string                      `gorm:"not null;column:account;size:100;default:'';index"`    // 账户地址
	Balance     decimal.Decimal             `gorm:"not null;column:balance;type:decimal(65,0);default:0"` // 余额
	TxHashs     datatypes.JSONSlice[string] `gorm:"column:tx_hashs"`                                      // null
}

func (RORTransferBalance) TableName() string {
	return "abt_ror_transfer"
}

type ListingPO struct {
	gorm.Model
	BusinessID uint64
	UserID     uint64 `gorm:"column:user_id"`
}

func (p *ListingPO) TableName() string {
	return "listing"
}

type User struct {
	gorm.Model
	Name string `gorm:"column:name;uniqueIndex;type:varchar(255)"`
	Age  int32
}
