package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

var verbose bool

// parseConfig parses input in 'key=value' format into a map.
// Empty lines and lines that start with '#' are ignored
func parseConfig(content io.Reader) (map[string]string, error) {
	r := make(map[string]string)
	lines := bufio.NewScanner(content)
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		toks := strings.SplitN(line, "=", 2)
		if len(toks) != 2 {
			return nil, fmt.Errorf("Invalid line '%s'", line)
		}
		key := strings.TrimSpace(toks[0])
		if len(key) == 0 {
			return nil, fmt.Errorf("Empty key in line '%s'", line)
		}
		value := strings.TrimSpace(toks[1])
		if len(value) == 0 {
			return nil, fmt.Errorf("Empty value in line '%s'", line)
		}
		r[key] = value
	}

	return r, lines.Err()
}

type config struct {
	vault              string
	awsAccessKeyId     string
	awsSecretAccessKey string
	proxy              string
	proxyPort          int
	region             string
	dbfileSize         int
}

func parseConfigFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseConfig(file)
}

// configFor reads directory specific configuration by merging configuration
// from the following files:
//
//  - dir/.glacier-backup/config
//  - ~/.glacier-backup files
func configFor(dir string) (config, error) {
	cfg := config{}
	usr, err := user.Current()
	if err != nil {
		return cfg, err
	}
	usrConfigFile := filepath.Join(usr.HomeDir, ".glacier-backup")
	usrConfig, err := parseConfigFile(usrConfigFile)
	if err != nil {
		return cfg, fmt.Errorf("Failed to parse user config file %s: %s", usrConfigFile, err)
	}
	dirConfigFile := filepath.Join(dir, ".glacier-backup", "config")
	dirConfig, err := parseConfigFile(dirConfigFile)
	if err != nil {
		return cfg, fmt.Errorf("Failed to parse dir config file %s: %s", dirConfigFile, err)
	}

	getConf := func(key string) string {
		v, ok := dirConfig[key]
		if ok {
			return v
		}
		v, ok = usrConfig[key]
		return v
	}

	cfg.vault = getConf("vault")
	cfg.awsAccessKeyId = getConf("aws_access_key_id")
	cfg.awsSecretAccessKey = getConf("aws_secret_access_key")
	cfg.proxy = getConf("proxy")
	proxyPort := getConf("proxy_port")
	if len(proxyPort) != 0 {
		cfg.proxyPort, err = strconv.Atoi(proxyPort)
		if err != nil {
			return cfg, fmt.Errorf("Invalid proxy_port value '%s'", proxyPort)
		}
	}
	cfg.region = getConf("region")
	dbfileSize := getConf("dbfile_size")
	if len(dbfileSize) != 0 {
		cfg.dbfileSize, err = strconv.Atoi(dbfileSize)
		if err != nil {
			return cfg, fmt.Errorf("Invalid dbfile_size value '%s'", dbfileSize)
		}
	} else {
		cfg.dbfileSize = 20
	}

	if verbose {
		log.Printf("Combined configuration: %+v\n", cfg)
	}

	return cfg, nil
}

func backup(dir string) error {
	log.Println("Backing up directory", dir)
	if _, err := configFor(dir); err != nil {
		return err
	}
	return nil
}

func compact(dir string) error {
	log.Println("Compacting db files in directory", dir)
	return nil
}

func main() {
	var compactFlag bool
	flag.BoolVar(&compactFlag, "compact", false, "Merge multiple db files into one")
	flag.BoolVar(&verbose, "v", false, "Be more verbose")
	flag.Parse()

	for _, dir := range flag.Args() {
		if err := backup(dir); err != nil {
			log.Printf("Failed to backup directory %s: %s\n", dir, err)
			continue
		}
		if compactFlag {
			if err := compact(dir); err != nil {
				log.Printf("Failed to compact db files in directory %s: %s", dir, err)
				continue
			}
		}
	}

	log.Println("DONE")
}
