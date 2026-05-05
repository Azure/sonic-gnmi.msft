package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers/boot_helpers"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type bootResponse struct {
	Current   string   `json:"current"`
	Next      string   `json:"next"`
	Available []string `json:"available"`
}

func getBoot(_ sdc.CmdArgs, _ sdc.OptionMap) ([]byte, error) {
	bl, err := helpers.DetectBootloader()
	if err != nil {
		log.Errorf("Failed to detect bootloader: %v", err)
		return nil, err
	}

	current, err := bl.GetCurrentImage()
	if err != nil {
		return nil, err
	}

	next, err := bl.GetNextImage()
	if err != nil {
		return nil, err
	}

	images, err := bl.GetInstalledImages()
	if err != nil {
		return nil, err
	}

	if images == nil {
		images = []string{}
	}

	resp := bootResponse{
		Current:   current,
		Next:      next,
		Available: images,
	}

	return json.Marshal(resp)
}
