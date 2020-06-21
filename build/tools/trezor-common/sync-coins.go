//usr/bin/go run $0 $@ ; exit
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	build "github.com/nbcorg/blockbook/build/tools"
)

const (
	configsDir          = "configs"
	trezorCommonDefsURL = "https://raw.githubusercontent.com/trezor/trezor-firmware/master/common/defs/bitcoin/"
)

type trezorCommonDef struct {
	Name                  string `json:"coin_name"`
	Shortcut              string `json:"coin_shortcut"`
	Label                 string `json:"coin_label"`
}

func getTrezorCommonDef(coin string) (*trezorCommonDef, error) {
	req, err := http.NewRequest("GET", trezorCommonDefsURL+coin+".json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Github request status code " + strconv.Itoa(resp.StatusCode))
	}
	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tcd trezorCommonDef
	json.Unmarshal(bb, &tcd)
	return &tcd, nil
}

func writeConfig(coin string, config *build.Config) error {
	path := filepath.Join(configsDir, "coins", coin+".json")
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	buf, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	n, err := out.Write(buf)
	if err != nil {
		return err
	}
	if n < len(buf) {
		return io.ErrShortWrite
	}
	return nil
}

func main() {
	var coins []string
	if len(os.Args) < 2 {
		filepath.Walk(filepath.Join(configsDir, "coins"), func(path string, info os.FileInfo, err error) error {
			n := strings.TrimSuffix(info.Name(), ".json")
			if n != info.Name() {
				coins = append(coins, n)
			}
			return nil
		})
	} else {
		coins = append(coins, os.Args[1])
	}
	for _, coin := range coins {
		config, err := build.LoadConfig(configsDir, coin)
		if err == nil {
			var tcd *trezorCommonDef
			tcd, err = getTrezorCommonDef(coin)
			if err == nil {
				if tcd.Name != "" {
					config.Coin.Name = tcd.Name
				}
				if tcd.Shortcut != "" {
					config.Coin.Shortcut = tcd.Shortcut
				}
				if tcd.Label != "" {
					config.Coin.Label = tcd.Label
				}
				err = writeConfig(coin, config)
				if err == nil {
					fmt.Printf("%v updated\n", coin)
				}
			}
		}
		if err != nil {
			fmt.Printf("%v update error %v\n", coin, err)
		}
	}
}
