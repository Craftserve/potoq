package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/ziutek/mymysql/godrv"

	"github.com/Craftserve/potoq/filters/cloudybans"
)

func main() {
	database, dberr := sql.Open("mymysql", os.Getenv("PROXY_MYSQL_URL"))
	if dberr != nil {
		panic(dberr)
	}
	dberr = database.Ping()
	if dberr != nil {
		panic(dberr)
	}

	sender := cloudybans.CommandSender{"console", HasPermission, SendChatMessage}
	handled := cloudybans.HandleCommand(sender, os.Args[1:])
	if !handled {
		log.Println("Unknown command")
	}
}

func HasPermission(perm string) bool {
	return true
}

func SendChatMessage(msg string) {
	log.Println(msg)
}
