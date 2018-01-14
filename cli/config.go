package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/naoina/toml"

	"github.com/nyaxt/otaru/util"
)

type CliConfig struct {
	Host map[string]Host
}

type Host struct {
	ApiEndpoint      string
	ExpectedCertFile string

	//ClientCertFile   string
}

func NewConfig(configdir string) (*CliConfig, error) {
	if err := util.IsDir(configdir); err != nil {
		return nil, fmt.Errorf("configdir \"%s\" is not a dir: %v", configdir, err)
	}
	os.Setenv("OTARUDIR", configdir)

	tomlpath := path.Join(configdir, "cliconfig.toml")

	buf, err := ioutil.ReadFile(tomlpath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %v", err)
	}

	cfg := CliConfig{}
	if err := toml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config file: %v", err)
	}

	// TODO expandpath *File

	return &cfg, nil
}
