package db

import (
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
)

var engine *xorm.Engine

// Init SQlite3 DB for discovery server in memory
// (since it's for examples and tests)
func Init() (err error) {

	engine, err = xorm.NewEngine("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return
	}

	engine.SetMaxIdleConns(30)
	engine.SetMaxOpenConns(30)
	engine.ShowSQL(true)

	if err = engine.Ping(); err != nil {
		return
	}

	return createTables()
}

// Close SQlite3 DB
func Close() (err error) {
	if engine != nil {
		err = engine.Close()
	}
	return
}

func createTables() (err error) {

	//
	// node table
	//

	const nodeTable = `CREATE TABLE node (
        id               INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
        [key]            CHAR (66),
        service_address  CHAR (50),
        location         CHAR (100),
        version          TEXT,
        priority         INTEGER,
        created          DATETIME,
        updated          DATETIME
    );`

	const nodeIndex = `CREATE UNIQUE INDEX idx_node_key ON node ( "key" );`

	if err = createTable("node", nodeTable, nodeIndex); err != nil {
		return
	}

	//
	// service table
	//

	const serviceTable = `CREATE TABLE service (
        id                   INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
        [key]                CHAR (66),
        address              CHAR (50),
        hide_from_discovery  INTEGER,
        allow_nodes          TEXT,
        version              CHAR (10),
        created              DATETIME,
        updated              DATETIME,
        node_id              INTEGER,

        FOREIGN KEY (node_id)
        REFERENCES  node (id) ON DELETE CASCADE
    );`

	const serviceIndex = `CREATE UNIQUE INDEX
        idx_service_key ON service ( "key" );`

	const serviceNodeIdIndex = `CREATE INDEX
        idx_service_node_id ON service ( "node_id" );`

	err = createTable("service", serviceTable,
		serviceIndex, serviceNodeIdIndex)

	if err != nil {
		return
	}

	//
	// attributes table
	//

	const attributesTable = `CREATE TABLE attributes (
        name        CHAR (20),
        service_id  INTEGER,

        FOREIGN KEY (service_id)
        REFERENCES  service (id) ON DELETE CASCADE
    );`

	const attributesNameIndex = `CREATE INDEX
        idx_attributes_name ON attributes ( "name" );`

	const attributesServiceIdIndex = `CREATE INDEX
        idx_attributes_service_id ON attributes ( "service_id" );`

	err = createTable("attributes",
		attributesTable,
		attributesNameIndex,
		attributesServiceIdIndex)

	return
}

func createTable(name, create string, indices ...string) (err error) {

	var exist bool

	if exist, err = engine.IsTableExist(name); err != nil {
		return
	}

	if exist == false {

		if _, err = engine.Exec(create); err != nil {
			return
		}

		for _, idx := range indices {
			if _, err = engine.Exec(idx); err != nil {
				return
			}
		}

	}

	return

}
