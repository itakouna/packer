package gridscale

import (
	"context"
	"fmt"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepCreateServer struct {
	serverID   string
	storageIDs []string
	ipID       string
}

func (s *stepCreateServer) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)
	sshKeyId := state.Get("ssh_key_id").(string)

	publicNetwork, err := client.GetNetworkPublic(context.Background())
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error getting publicNetwork: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Creating server...")
	server, err := client.CreateServer(
		context.Background(),
		gsclient.ServerCreateRequest{
			Name:   c.ServerName,
			Cores:  c.ServerCores,
			Memory: c.ServerMemory,
		})
	if err != nil {
		err := fmt.Errorf("Error creating server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.serverID = server.ObjectUUID
	state.Put("server_id", server.ObjectUUID)

	ui.Say("Creating bootable storage...")
	var sshkeys []string

	storage, err := client.CreateStorage(
		context.Background(),
		gsclient.StorageCreateRequest{
			Capacity:    c.StorageCapacity,
			Name:        c.ServerName,
			StorageType: gsclient.InsaneStorageType,
			Template: &gsclient.StorageTemplate{
				Password:     c.Password,
				PasswordType: gsclient.PlainPasswordType,
				Hostname:     c.Hostname,
				Sshkeys:      append(sshkeys, sshKeyId),
				TemplateUUID: c.TemplateUUID,
			},
		})

	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error creating storage: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	s.storageIDs = append(s.storageIDs, storage.ObjectUUID)
	state.Put("storage_id", storage.ObjectUUID)

	ui.Say("Linking server with bootable storage...")
	err = client.LinkStorage(context.Background(), server.ObjectUUID, storage.ObjectUUID, true)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error linking Server with Storage: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if c.SecondaryStorage == true {
		ui.Say("Creating unbootable storage...")
		storage, err := client.CreateStorage(
			context.Background(),
			gsclient.StorageCreateRequest{
				Capacity:    c.StorageCapacity,
				Name:        c.ServerName,
				StorageType: gsclient.InsaneStorageType,
			})

		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error creating storage: %s", err))
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		s.storageIDs = append(s.storageIDs, storage.ObjectUUID)
		state.Put("storage_id", storage.ObjectUUID)

		ui.Say("Linking server with unbootable storage...")
		err = client.LinkStorage(context.Background(), server.ObjectUUID, storage.ObjectUUID, false)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error linking Server with Storage: %s", err))
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

	}
	ui.Say("Creating IP...")
	ip, err := client.CreateIP(
		context.Background(),
		gsclient.IPCreateRequest{
			Name:   c.ServerName,
			Family: gsclient.IPv4Type,
		})
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error creating IP: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.ipID = ip.ObjectUUID
	state.Put("server_ip", ip.IP)

	ui.Say("Linking server with IP...")
	err = client.LinkIP(context.Background(), server.ObjectUUID, ip.ObjectUUID)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error linking Server with IP: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Linking Server with PublicNetwork...")
	err = client.LinkNetwork(context.Background(), server.ObjectUUID, publicNetwork.Properties.ObjectUUID, "", false, 0, nil, nil)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error linking Server with PublicNetwork: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if c.IsoImageUUID != "" {
		ui.Say("Linking Server with ISO-image...")
		serverIsoImageRelationCreateRequest := gsclient.ServerIsoImageRelationCreateRequest{
			ObjectUUID: c.IsoImageUUID,
		}
		err = client.CreateServerIsoImage(context.Background(), server.ObjectUUID, serverIsoImageRelationCreateRequest)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error linking Server with ISO-image: %s", err))
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}
	ui.Say("Starting Server...")
	err = client.StartServer(context.Background(), server.ObjectUUID)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error starting Server: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	return multistep.ActionContinue
}

func (s *stepCreateServer) Cleanup(state multistep.StateBag) {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)

	if s.serverID != "" {
		ui.Say("Shutdown Server...")
		err := client.StopServer(context.Background(), s.serverID)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error Shutdown server: %s", err))
		}
	}

	if s.serverID != "" {
		ui.Say("Destroying server...")
		err := client.DeleteServer(context.Background(), s.serverID)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error destroying server. Please destroy it manually: %s", err))
		}
	}

	for _, id := range s.storageIDs {
		ui.Say("Destroying storage...")
		err := client.DeleteStorage(context.Background(), id)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error destroying storage. Please destroy it manually: %s", err))
		}
	}

	if s.ipID != "" {
		ui.Say("Destroying IP...")
		err := client.DeleteIP(context.Background(), s.ipID)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error destroying ip. Please destroy it manually: %s", err))
		}
	}

	return

}
