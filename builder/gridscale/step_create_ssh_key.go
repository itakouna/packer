package gridscale

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/gridscale/gsclient-go"
	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/packer/common/uuid"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepCreateSSHKey struct {
	Debug        bool
	DebugKeyPath string
	keyId        string
}

func (s *stepCreateSSHKey) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)

	ui.Say("Creating temporary ssh key for server...")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	priv_der := x509.MarshalPKCS1PrivateKey(priv)
	priv_blk := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   priv_der,
	}

	c.Comm.SSHPrivateKey = pem.EncodeToMemory(&priv_blk)
	pub, _ := ssh.NewPublicKey(&priv.PublicKey)

	resp, err := client.CreateSshkey(context.Background(), gsclient.SshkeyCreateRequest{
		Name:   fmt.Sprintf("packer-%s", uuid.TimeOrderedUUID()),
		Sshkey: string(bytes.Trim(ssh.MarshalAuthorizedKey(pub), "\n")),
	})
	if err != nil {
		err := fmt.Errorf("Error getting temporary SSH key: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.keyId = resp.ObjectUUID
	state.Put("ssh_key_id", resp.ObjectUUID)

	if s.Debug {
		ui.Message(fmt.Sprintf("Saving key for debug purposes: %s", s.DebugKeyPath))
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			state.Put("error", fmt.Errorf("Error saving debug key: %s", err))
			return multistep.ActionHalt
		}
		defer f.Close()

		// Write the key out
		if _, err := f.Write(pem.EncodeToMemory(&priv_blk)); err != nil {
			state.Put("error", fmt.Errorf("Error saving debug key: %s", err))
			return multistep.ActionHalt
		}

		// Chmod it so that it is SSH ready
		if runtime.GOOS != "windows" {
			if err := f.Chmod(0600); err != nil {
				state.Put("error", fmt.Errorf("Error setting permissions of debug key: %s", err))
				return multistep.ActionHalt
			}
		}
	}

	return multistep.ActionContinue
}

func (s *stepCreateSSHKey) Cleanup(state multistep.StateBag) {
	if s.keyId == "" {
		return
	}

	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Deleting temporary ssh key...")
	err := client.DeleteSshkey(context.Background(), s.keyId)
	if err != nil {
		log.Printf("Error cleaning up ssh key: %s", err)
		ui.Error(fmt.Sprintf(
			"Error cleaning up ssh key. Please delete the key manually: %s", err))
	}
}
