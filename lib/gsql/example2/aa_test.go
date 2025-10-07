package example2

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	gsql "github.com/donutnomad/gotoolkit/lib/gsql"
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"github.com/samber/lo"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestD(t *testing.T) {
	// 先连接 MySQL 服务器(不指定数据库)
	dsnWithoutDB := "root:123456@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
	_db, err := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("连接 MySQL 服务器失败: %v", err)
	}

	// 检查并创建数据库
	dbName := "test"
	createDBSQL := "CREATE DATABASE IF NOT EXISTS `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"
	if err := _db.Exec(createDBSQL).Error; err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	t.Logf("数据库 %s 已就绪", dbName)

	// 重新连接到指定数据库
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	_db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	//
	err = _db.AutoMigrate(&User{}, &ListingPO{}, &RORRequest{}, &RORTransferBalance{})
	if err != nil {
		t.Fatal(err)
	}

	db := gsql.NewDefaultGormDB(_db)

	//db.Create(&User{
	//	Name: "张三",
	//	Age:  18,
	//})
	//db.Create(&User{
	//	Name: "li",
	//	Age:  22,
	//})

	// SELECT * FROM `users` WHERE name='li' AND `users`.`deleted_at` IS NULL
	//var users []User
	//db.Model(&User{}).Where("name=?", "li").Find(&users)
	//
	//spew.Dump(users)

	//var users2 []User
	//var u = UserTable.As("u")
	//err = NewQuery().
	//	Select(u.Name).
	//	From(u).
	//	Where(u.Name.Not("张三")).
	//	Find(db, &users2)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//spew.Dump(users2)

	//{
	//	var listings []ListingPO
	//
	//	var u = UserTable.As("token_sale")
	//	var l = ListingTable.As("l")
	//
	//	var optBizID = mo.Some(uint64(123))
	//
	//	err = NewQuery().
	//		Select(l.BusinessID, l.UserID).
	//		From(l).
	//		Join(LeftJoin(u).On(u.ID.EqF(l.UserID))).
	//		Where(
	//			l.BusinessID.EqOpt(optBizID),
	//			Or(
	//				l.BusinessID.In(1, 2, 3),
	//				l.UserID.Eq(123),
	//			),
	//		).
	//		Find(db, &listings)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	spew.Dump(listings)
	//}

	// 更加复杂的查询

	{
		var u = UserSchema
		//u.Name.Set(u.Name.As("name2"))
		//spew.Dump(u.Name)
		//uName := u.Name.As("name2")
		//uName := u.Name
		//var l = ListingTable.As("l")

		type tmpRet struct {
			Name string
			ID   uint
		}

		userNameTable := gsql.DefineTempTable[tmpRet](
			struct {
				Name field.Pattern[string]
				ID   field.Comparable[uint]
			}{
				Name: u.Name,
				ID:   u.ID,
			},
			gsql.Select(u.Name, u.ID).From(u),
		)
		_ = userNameTable
		userNameTable2 := gsql.DefineTempTable[uint](
			struct {
				Name field.Pattern[string]
				ID   field.Comparable[uint]
			}{
				Name: u.Name,
				ID:   u.ID,
			},
			gsql.Select(u.Name, u.ID).From(u),
		)
		_ = userNameTable2

		//var rets []tmpRet

		//rets, err := scopes8.Pluck(db, gsql.SelectG[string]().
		//	From(userNameTable2).
		//	Where(
		//	//gsql.Or(
		//	//	userNameTable.Fields.Name.Eq("张三"),
		//	//	userNameTable.Fields.Name.LikeOpt(mo.Some("%li%")),
		//	//),
		//	), userNameTable2.Fields.Name)
		//spew.Dump(rets)
		//spew.Dump(err)

		//rets, err := gsql.SelectG[string](
		//	userNameTable.Fields.Name,
		//).
		//	From(userNameTable).
		//	Where(
		//	//gsql.Or(
		//	//	userNameTable.Fields.Name.Eq("张三"),
		//	//	userNameTable.Fields.Name.LikeOpt(mo.Some("%li%")),
		//	//),
		//	).
		//	Find(db)
		////Pluck(db, userNameTable.Fields.Name)
		//spew.Dump(lo.FromSlicePtr(rets))
		//spew.Dump(err)

		spew.Dump(gsql.PluckG(u.Age).From(u).Where(u.Age.Eq(12)).Distinct().Find(db))

		//var rets []string
		//err := gsql.SelectG[tmpRet]().
		//	From(userNameTable).
		//	Where(
		//	//gsql.Or(
		//	//	userNameTable.Fields.Name.Eq("张三"),
		//	//	userNameTable.Fields.Name.LikeOpt(mo.Some("%li%")),
		//	//),
		//	).
		//	Pluck(db, userNameTable.Fields.Name, &rets)
		////Take(db)
		//if err != nil {
		//	t.Fatal(err)
		//}
		//spew.Dump(rets)

		//tx := db.Save(&User{
		//	Model: gorm.Model{
		//		ID: 2,
		//	},
		//	Name: "bb",
		//	Age:  12,
		//})
		//if tx.Error != nil {
		//	t.Fatal(tx.Error)
		//}
		//fmt.Println(tx.RowsAffected)

		//err = gsql.InsertInto(UserTable).Value(&User{
		//	Model: gorm.Model{
		//		ID: 2,
		//	},
		//	Name: "aa",
		//	Age:  12,
		//}).DuplicateUpdate().Exec(db)
		//if err != nil {
		//	t.Fatal(err)
		//}

		return

		//err = gsql.InsertInto(UserTable).Value(&User{
		//	Model: gorm.Model{
		//		ID: 1,
		//	},
		//	Name: "aa",
		//	Age:  8,
		//}).DuplicateUpdate().Exec(db)
		//if err != nil {
		//	t.Fatal(err)
		//}

		//err = gsql.InsertInto(UserTable,
		//	UserTable.CreatedAt,
		//	UserTable.UpdatedAt,
		//	UserTable.DeletedAt,
		//	UserTable.Name,
		//	UserTable.Age,
		//).Select(
		//	gsql.Select(
		//		UserTable.CreatedAt,
		//		UserTable.UpdatedAt,
		//		UserTable.DeletedAt,
		//		UserTable.Name,
		//		UserTable.Age,
		//	).From(UserTable),
		//).DuplicateUpdate(UserTable.Name).Exec(db)
		//if err != nil {
		//	t.Fatal(err)
		//}

		//row, err := gsql.SelectG[User]().From(UserTable).Where(UserTable.ID.Eq(1)).Unscoped().Delete(db)
		//if err != nil {
		//	t.Fatal(err)
		//}
		//fmt.Println("删除结果:", row)
		//
		//rets2, err := gsql.SelectG[User]().From(UserTable).Unscoped().Find(db)
		//if err != nil {
		//	t.Fatal(err)
		//}
		//spew.Dump(rets2)
		//
		//return
	}

	{
		ror := RORRequestSchema.As("rrr")
		//r2 := gsql.DefineTable("r2", ror, gsql.Select(
		//	ror.ID,
		//	ror.CreatedAt,
		//	ror.UpdatedAt,
		//	ror.NFTContract,
		//	ror.NFTID,
		//	ror.NFTStatus,
		//	ror.Updated,
		//	ror.TokenID,
		//	ror.TokenName,
		//	ror.TokenSymbol,
		//	ror.TokenDecimals,
		//	ror.Creator,
		//	ror.From,
		//	ror.Amount,
		//	ror.Receiver,
		//	ror.PartyA,
		//	ror.ExecutionDateStartTime,
		//	ror.ExecutionDateEndTime,
		//	ror.ExecutionDateDay,
		//	ror.ExecutionDateType,
		//	ror.LogicAnd,
		//	ror.RecordCreatedAt,
		//	ror.NFTCreatedAt,
		//	ror.Status,
		//	ror.UpdateAtBlockNumber,
		//	ror.UpdateAtBlockTimestamp,
		//).From(ror))

		tmp2 := gsql.DefineTempTableAny(
			struct {
				RORRequestSchemaType
				Rn field.Comparable[uint64]
			}{
				RORRequestSchemaType: RORRequestSchema,
				Rn:                   field.NewComparable[uint64]("", "rn"),
			},
			gsql.
				Select(gsql.Star, gsql.Field("ROW_NUMBER() OVER (PARTITION BY nft_id ORDER BY created_at) as rn")).
				From(ror),
		)

		tmp1 := gsql.DefineTempTableAny(
			RORRequestSchema,
			gsql.Select().From(tmp2).Where(tmp2.Fields.Rn.Eq(1)),
		)

		fmt.Println("打印tmp1:", lo.ToPtr(tmp1.Query()).ToSQL())

		nt := RORTransferBalanceSchema.As("rt")

		//var ret []RORTransferBalance
		ret, err := gsql.
			SelectG[RORTransferBalance](
			nt.CreatedAt,
			nt.UpdatedAt,
			nt.Balance,
			nt.Account,
		).
			From(nt).
			Join(gsql.InnerJoin(tmp1).
				On(nt.NFTID.EqF(tmp1.Fields.NFTID)).
				And(nt.NFTContract.EqF(tmp1.Fields.NFTContract)),
			).
			//Order(nt.CreatedAt, false).
			Offset(1).
			Limit(20).
			First(db)
		if err != nil {
			t.Fatal(err)
		}

		spew.Dump(ret)
		// tmp2.nft_id = nt.nft_id AND tmp2.nft_contract = nt.nft_contract
		//fmt.Println(sq1.ToSQL(db))
		//fmt.Println("哇哈哈哈")
		//fmt.Println(sq2.ToSQL())

		//	var subQuery = `
		//SELECT
		//FROM ` + rorTableName + ` as ror
		//
		//UNION ALL
		//
		//SELECT
		//	tmp2.id,
		//	nt.created_at,
		//	nt.updated_at,
		//	tmp2.nft_contract,
		//	tmp2.nft_id,
		//	tmp2.nft_status,
		//	tmp2.updated,
		//	tmp2.token_id,
		//	tmp2.token_name,
		//	tmp2.token_symbol,
		//	tmp2.token_decimals,
		//	tmp2.creator,
		//	tmp2.party_a as ` + "`from`" + `,
		//	nt.balance as amount,
		//	nt.account as receiver,
		//	tmp2.party_a,
		//	tmp2.execution_date_start_time,
		//	tmp2.execution_date_end_time,
		//	tmp2.execution_date_day,
		//	tmp2.execution_date_type,
		//	tmp2.logic_and,
		//	tmp2.conditions,
		//	tmp2.record_created_at,
		//	tmp2.nft_created_at,
		//	tmp2.status,
		//	tmp2.update_at_block_number,
		//	tmp2.update_at_block_timestamp
		//FROM ` + model.RORTransferBalance{}.TableName() + ` as nt
		//INNER JOIN (
		//	SELECT *
		//	FROM (
		//		SELECT *, ROW_NUMBER() OVER (PARTITION BY nft_id ORDER BY created_at) as rn FROM ` + rorTableName + `
		//	) tmp1
		//	WHERE tmp1.rn = 1
		//) tmp2 ON tmp2.nft_id = nt.nft_id AND tmp2.nft_contract = nt.nft_contract
		//`

	}

	//s := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
	//	var all []map[string]any
	//	return tx.Select("name", "age").Table("aaa").Find(&all)
	//})
	//t.Log(s)
}
