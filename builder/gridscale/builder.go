package gridscale

import (
	"context"
	"fmt"
	"log"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

// The unique id for the builder
const BuilderId = "packer.gridscale"

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	c, warnings, errs := NewConfig(raws...)
	if errs != nil {
		return warnings, errs
	}
	b.config = *c

	return nil, nil
}

func (b *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	client := gsclient.NewClient(gsclient.DefaultConfiguration(b.config.APIKey, b.config.APIToken))

	// Set up the state
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("client", client)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Build the steps
	steps := []multistep.Step{
		&stepCreateSSHKey{
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("gs_%s.pem", b.config.PackerBuildName),
		},
		new(stepCreateServer),
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      communicator.CommHost(b.config.Comm.SSHHost, "server_ip"),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		&common.StepProvision{},
	}

	// Run the steps
	b.runner = common.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	if _, ok := state.GetOk("snapshot_name"); !ok {
		log.Println("Failed to find snapshot_name in state. Bug?")
		return nil, nil
	}

	artifact := &Artifact{
		SnapshotName: state.Get("snapshot_name").(string),
		SnapshotId:   state.Get("snapshot_image_id").(string),
		RegionNames:  state.Get("regions").([]string),
		Client:       client,
	}

	return artifact, nil
}
