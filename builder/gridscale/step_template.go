package gridscale

import (
	"context"
	"fmt"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepTemplate struct {
}

func (s *stepTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)
	storageID := state.Get("storage_id").(string)

	ui.Say(fmt.Sprintf("Creating snapshot: %v", c.TemplateName))
	snapshot, err := client.CreateStorageSnapshot(
		context.Background(),
		storageID,
		gsclient.StorageSnapshotCreateRequest{
			Name: c.TemplateName,
		})

	if err != nil {
		err := fmt.Errorf("Error creating snapshot: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Creating template: %v", c.TemplateName))
	template, err := client.CreateTemplate(
		context.Background(),
		gsclient.TemplateCreateRequest{
			Name:         c.TemplateName,
			SnapshotUUID: snapshot.ObjectUUID,
		})

	if err != nil {
		err := fmt.Errorf("Error creating template: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	ui.Say(fmt.Sprintf("Created template %v with uuid: %v", c.TemplateName, template.ObjectUUID))

	return multistep.ActionContinue
}

func (s *stepTemplate) Cleanup(state multistep.StateBag) {
	// no cleanup
}
