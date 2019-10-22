package gridscale

import (
	"errors"
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/common/uuid"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`
	// The client TOKEN to use to access your account.

	APIToken     string `mapstructure:"api_token" required:"true"`
	APIKey       string `mapstructure:"api_key" required:"true"`
	TemplateName string `mapstructure:"template_name" required:"false"`

	Hostname        string `mapstructure:"hostname" required:"false"`
	Password        string `mapstructure:"password" required:"false"`
	ServerName      string `mapstructure:"server_name" required:"false"`
	ServerCores     int    `mapstructure:"server_cores" required:"true"`
	ServerMemory    int    `mapstructure:"server_memory" required:"true"`
	StorageCapacity int    `mapstructure:"storage_capacity" required:"true"`
	TemplateUUID    string `mapstructure:"template_uuid" required:"true"`

	Tags []string `mapstructure:"tags" required:"false"`

	ctx interpolate.Context
}

func NewConfig(raws ...interface{}) (*Config, []string, error) {
	c := new(Config)

	var md mapstructure.Metadata
	err := config.Decode(c, &config.DecodeOpts{
		Metadata:           &md,
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"run_command",
			},
		},
	}, raws...)
	if err != nil {
		return nil, nil, err
	}

	// Defaults
	if c.APIToken == "" {
		// Default to environment variable for api_token, if it exists
		c.APIToken = os.Getenv("GRIDSCALE_TOKEN")
	}

	if c.APIKey == "" {
		c.APIKey = os.Getenv("GRIDSCALE_UUID")
	}

	if c.TemplateName == "" {
		def, err := interpolate.Render("packer-{{timestamp}}", nil)
		if err != nil {
			panic(err)
		}

		// Default to packer-{{ unix timestamp (utc) }}
		c.TemplateName = def
	}

	if c.ServerName == "" {
		// Default to packer-[time-ordered-uuid]
		c.ServerName = fmt.Sprintf("packer-%s", uuid.TimeOrderedUUID())
	}

	var errs *packer.MultiError
	if es := c.Comm.Prepare(&c.ctx); len(es) > 0 {
		errs = packer.MultiErrorAppend(errs, es...)
	}
	if c.APIToken == "" {
		// Required configurations that will display errors if not set
		errs = packer.MultiErrorAppend(
			errs, errors.New("api_token for auth must be specified"))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, nil, errs
	}

	packer.LogSecretFilter.Set(c.APIToken)
	return c, nil, nil
}
