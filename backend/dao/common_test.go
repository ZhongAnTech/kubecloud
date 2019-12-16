package dao

import (
	"fmt"
	"testing"

	_ "kubecloud/test/mock"

	"github.com/astaxie/beego/orm"
	"github.com/stretchr/testify/assert"
)

func TestGetOrmer(t *testing.T) {
	ormer1 := GetOrmer()
	assert.NotNil(t, ormer1, "should not nil")
	ormer2 := GetOrmer()
	assert.Equal(t, ormer1, ormer2, "should equal")
}

func TestTransactional(t *testing.T) {
	type Row struct {
		Id      int64  `orm:"column(id)"`
		Name    string `orm:"column(name)"`
		SubName string `orm:"column(subname)"`
	}
	txNames := []string{"a", "b", "c", "d", "e"}
	txSubSize := 10
	querySql := "select tx.id as id, tx.name as name, tx_sub.name as subname from tx right join tx_sub on tx.id = tx_sub.pid"
	makeSubName := func(name string, i int) string {
		return fmt.Sprintf("%v_%v", name, i)
	}
	setupTables := func(ormer orm.Ormer) (err error) {
		sqls := []string{
			`drop table if exists tx;`,
			`create table tx (
				id bigint not null auto_increment,
				name varchar(255) not null,
				primary key (id),
				index (name)
			);`,
			`drop table if exists tx_sub;`,
			`create table tx_sub (
				id bigint not null auto_increment,
				pid bigint not null,
				name varchar(255) not null,
				primary key (id),
				index (pid),
				index (name)
			);`,
		}
		for _, sql := range sqls {
			_, err = ormer.Raw(sql).Exec()
		}
		return
	}

	t.Run("ShouldCommit", func(t *testing.T) {
		ormer := orm.NewOrm()
		assert.Nil(t, setupTables(ormer))
		actualRows := []Row{}
		expectedCount := int64(len(txNames)) * int64(txSubSize)
		actualCount, err := ormer.Raw(querySql).QueryRows(&actualRows)
		assert.Nil(t, err)
		assert.Equal(t, expectedCount, actualCount, "invalid rows")
		for i := range txNames {
			for j := 0; j < txSubSize; j++ {
				expected := Row{
					Id:      int64(i) + 1,
					Name:    txNames[i],
					SubName: makeSubName(txNames[i], j+1),
				}
				actual := actualRows[i*txSubSize+j]
				assert.Equal(t, expected, actual, "invalid row")
			}
		}
	})

	t.Run("ShouldRollback", func(t *testing.T) {
		ormer := orm.NewOrm()
		assert.Nil(t, setupTables(ormer))
		expectedErr := fmt.Errorf("oops")
		assert.Equal(t, expectedErr, actualErr, "should rollback with error")
		var expectedCount int64
		var actualCount int64
		err := ormer.Raw("select count(*) from tx").QueryRow(&actualCount)
		assert.Nil(t, err)
		assert.Equal(t, expectedCount, actualCount, "should be empty set")
	})
}
