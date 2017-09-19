package main

import (
	"encoding/json"
	"fmt"
	"github.com/th3osmith/gomark"
	"io/ioutil"
	"os"
	"path"
)

type config struct {
	UseTLS      bool
	Certificate string
	Key         string
	Port        int
	DbFile      string
	Username    string
	Password    string
	YoutubeKey  string
}

func getDefaultConfig() config {

	return config{
		false,
		"",
		"",
		3000,
		"",
		"",
		"",
		"",
	}
}

func readConfig(configFile string) (c config) {

	c = getDefaultConfig()

	f, err := ioutil.ReadFile(configFile)
	checkFatal(err, "Reading Config")

	if len(f) == 0 {
		return c
	}

	err = json.Unmarshal(f, &c)
	checkFatal(err, "Reading Config")

	return
}

type auther struct {
	username string
	password string
}

func (a *auther) CheckCredentials(username string, password string) bool {

	if len(a.username) == 0 && len(a.password) == 0 {
		return true
	}

	return username == a.username && password == a.password
}

func main() {

	configFile := os.Getenv("GOMARK_CONFIG")
	if configFile == "" {
		configFile = "/etc/gomark/config.json"
	}

	c := readConfig(configFile)

	home := os.Getenv("HOME")

	if len(c.DbFile) == 0 {
		c.DbFile = home + "/.gomark/db.json"
	}

	err := checkDbFile(c.DbFile)
	checkFatal(err, "Checking DB File")

	db, err := gomark.NewDatabaseFromFile(c.DbFile)
	checkFatal(err, "Creating DB")

	var server gomark.Server
	auth := &auther{c.Username, c.Password}

	config := gomark.HttpConfig{
		UseTLS:          c.UseTLS,
		CertificateFile: c.Certificate,
		KeyFile:         c.Key,
		Authenticator:   auth,
	}

	if c.YoutubeKey != "" {
		gomark.YoutubeKey = c.YoutubeKey
	}

	fmt.Printf("Gomark Sever starting on port %v\n", c.Port)
	gomark.ServeHttp(db, &server, c.Port, config)

}

func checkFatal(err error, context string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%v] %v", context, err)
		os.Exit(1)
	}
}

func checkDbFile(pathString string) error {

	dir := path.Dir(pathString)

	// Create the directory if not existent
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			return err
		}
	}

	// Create the file if not existent
	if _, err := os.Stat(pathString); os.IsNotExist(err) {
		file, err := os.Create(pathString)
		if err != nil {
			return err
		}
		file.Close()
	}

	return nil
}
