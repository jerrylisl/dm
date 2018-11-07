package relay

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/pingcap/check"
	"github.com/siddontang/go-mysql/mysql"

	"github.com/pingcap/tidb-enterprise-tools/pkg/gtid"
)

var _ = Suite(&testRelaySuite{})

func TestSuite(t *testing.T) {
	TestingT(t)
}

type testRelaySuite struct {
}

type MetaTestCase struct {
	uuid           string
	uuidWithSuffix string
	pos            mysql.Position
	gset           gtid.Set
}

func (r *testRelaySuite) TestLocalMeta(c *C) {
	dir, err := ioutil.TempDir("", "test_local_meta")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	gset0, _ := gtid.ParserGTID("mysql", "")
	gset1, _ := gtid.ParserGTID("mysql", "85ab69d1-b21f-11e6-9c5e-64006a8978d2:1-12")
	gset2, _ := gtid.ParserGTID("mysql", "85ab69d1-b21f-11e6-9c5e-64006a8978d2:13-23")
	gset3, _ := gtid.ParserGTID("mysql", "85ab69d1-b21f-11e6-9c5e-64006a8978d2:24-33")
	gset4, _ := gtid.ParserGTID("mysql", "85ab69d1-b21f-11e6-9c5e-64006a8978d2:34-46")
	gset5, _ := gtid.ParserGTID("mysql", "85ab69d1-b21f-11e6-9c5e-64006a8978d2:45-56")

	cases := []MetaTestCase{
		{
			uuid:           "server-a-uuid",
			uuidWithSuffix: "server-a-uuid.000001",
			pos:            mysql.Position{Name: "mysql-bin.000003", Pos: 123},
			gset:           gset1,
		},
		{
			uuid:           "server-b-uuid",
			uuidWithSuffix: "server-b-uuid.000002",
			pos:            mysql.Position{Name: "mysql-bin.000001", Pos: 234},
			gset:           gset2,
		},
		{
			uuid:           "server-b-uuid", // server-b-uuid again
			uuidWithSuffix: "server-b-uuid.000003",
			pos:            mysql.Position{Name: "mysql-bin.000002", Pos: 345},
			gset:           gset3,
		},
		{
			uuid:           "server-c-uuid",
			uuidWithSuffix: "server-c-uuid.000004",
			pos:            mysql.Position{Name: "mysql-bin.000004", Pos: 678},
			gset:           gset4,
		},
	}

	// load, but empty
	lm := NewLocalMeta("mysql", dir)
	err = lm.Load()
	c.Assert(err, IsNil)

	uuid, pos := lm.Pos()
	c.Assert(uuid, Equals, "")
	c.Assert(pos, DeepEquals, minCheckpoint)
	uuid, gset := lm.GTID()
	c.Assert(uuid, Equals, "")
	c.Assert(gset, DeepEquals, gset0)

	err = lm.Save(minCheckpoint, nil)
	c.Assert(err, NotNil)

	err = lm.Flush()
	c.Assert(err, NotNil)

	dirty := lm.Dirty()
	c.Assert(dirty, IsFalse)

	for _, cs := range cases {
		err = lm.AddDir(cs.uuid, nil, nil)
		c.Assert(err, IsNil)

		err = lm.Save(cs.pos, cs.gset)
		c.Assert(err, IsNil)

		currentUUID, pos2 := lm.Pos()
		c.Assert(currentUUID, Equals, cs.uuidWithSuffix)
		c.Assert(pos2, DeepEquals, cs.pos)

		currentUUID, gset = lm.GTID()
		c.Assert(currentUUID, Equals, cs.uuidWithSuffix)
		c.Assert(gset, DeepEquals, cs.gset)

		dirty = lm.Dirty()
		c.Assert(dirty, IsTrue)

		currentDir := lm.Dir()
		c.Assert(strings.HasSuffix(currentDir, cs.uuidWithSuffix), IsTrue)
	}

	err = lm.Flush()
	c.Assert(err, IsNil)

	dirty = lm.Dirty()
	c.Assert(dirty, IsFalse)

	// create a new LocalMeta, and load it
	lm2 := NewLocalMeta("mysql", dir)
	err = lm2.Load()
	c.Assert(err, IsNil)

	lastCase := cases[len(cases)-1]

	uuid, pos = lm2.Pos()
	c.Assert(uuid, Equals, lastCase.uuidWithSuffix)
	c.Assert(pos, DeepEquals, lastCase.pos)
	uuid, gset = lm2.GTID()
	c.Assert(uuid, Equals, lastCase.uuidWithSuffix)
	c.Assert(gset, DeepEquals, lastCase.gset)

	// another case for AddDir, specify pos and GTID
	cs := MetaTestCase{
		uuid:           "server-c-uuid",
		uuidWithSuffix: "server-c-uuid.000005",
		pos:            mysql.Position{Name: "mysql-bin.000005", Pos: 789},
		gset:           gset5,
	}
	err = lm.AddDir(cs.uuid, &cs.pos, cs.gset)
	c.Assert(err, IsNil)

	dirty = lm.Dirty()
	c.Assert(dirty, IsFalse)

	currentUUID, pos := lm.Pos()
	c.Assert(currentUUID, Equals, cs.uuidWithSuffix)
	c.Assert(pos, DeepEquals, cs.pos)

	currentUUID, gset = lm.GTID()
	c.Assert(currentUUID, Equals, cs.uuidWithSuffix)
	c.Assert(gset, DeepEquals, cs.gset)

	currentDir := lm.Dir()
	c.Assert(strings.HasSuffix(currentDir, cs.uuidWithSuffix), IsTrue)
}