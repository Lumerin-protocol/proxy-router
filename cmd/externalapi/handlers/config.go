package handlers

import (
	"context"
	"errors"
	"log"

	configv1 "github.com/Lumerin-protocol/lumerin-sdk-go/proto/config/v1"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

type handlers struct {
	repo *msgdata.ConfigInfoRepo
}

func NewConfigHandlers(repo *msgdata.ConfigInfoRepo) configv1.ConfigsServiceServer {
	return &handlers{
		repo: repo,
	}
}

func (s *handlers) GetConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.GetConfigResponse, error) {
	result, err := s.repo.GetConfigInfo(req.ID)
	if err != nil {
		return nil, errors.New("not found")
	}
	return &configv1.GetConfigResponse{
		Config: s.mapConfigToProto(result),
	}, nil
}

func (s *handlers) GetConfigs(context.Context, *configv1.GetConfigsRequest) (*configv1.GetConfigsResponse, error) {
	results := s.repo.GetAllConfigInfos()

	return &configv1.GetConfigsResponse{
		Configs: s.mapConfigsToProto(results),
	}, nil
}

func (s *handlers) CreateConfig(ctx context.Context, req *configv1.CreateConfigRequest) (*configv1.CreateConfigResponse, error) {
	config := s.mapProtoToConfig(req.Config)
	s.repo.AddConfigInfo(config)

	configMsg := msgdata.ConvertConfigInfoJSONtoConfigInfoMSG(config)
	_, err := s.repo.Ps.PubWait(msgbus.ConfigMsg, msgbus.IDString(configMsg.ID), configMsg)
	if err != nil {
		log.Printf("Config POST request failed to update msgbus: %s", err)
	}

	return &configv1.CreateConfigResponse{}, nil
}

func (s *handlers) DeleteConfig(ctx context.Context, req *configv1.DeleteConfigRequest) (*configv1.DeleteConfigResponse, error) {
	if err := s.repo.DeleteConfigInfo(req.ID); err != nil {
		return nil, errors.New("not Found")
	}

	_, err := s.repo.Ps.UnpubWait(msgbus.ConfigMsg, msgbus.IDString(req.ID))
	if err != nil {
		log.Printf("Config DELETE request failed to update msgbus: %s", err)
	}
	return &configv1.DeleteConfigResponse{}, nil
}

func (s *handlers) UpdateConfig(ctx context.Context, req *configv1.UpdateConfigRequest) (*configv1.UpdateConfigResponse, error) {
	config := s.mapProtoToConfig(req.Config)

	if err := s.repo.UpdateConfigInfo(config.ID, config); err != nil {
		return nil, errors.New("not found")
	}

	configMsg := msgdata.ConvertConfigInfoJSONtoConfigInfoMSG(config)
	_, err := s.repo.Ps.SetWait(msgbus.ConfigMsg, msgbus.IDString(config.ID), configMsg)
	if err != nil {
		log.Printf("Config PUT request failed to update msgbus: %s", err)
	}

	return &configv1.UpdateConfigResponse{}, nil
}

func (s *handlers) mapConfigToProto(config msgdata.ConfigInfoJSON) *configv1.Config {
	return &configv1.Config{
		ID:           config.ID,
		DefaultDest:  config.DefaultDest,
		NodeOperator: config.NodeOperator,
	}
}

func (s *handlers) mapConfigsToProto(configs []msgdata.ConfigInfoJSON) []*configv1.Config {
	protos := make([]*configv1.Config, len(configs))

	for i, result := range configs {
		protos[i] = s.mapConfigToProto(result)
	}

	return protos
}

func (h *handlers) mapProtoToConfig(proto *configv1.Config) msgdata.ConfigInfoJSON {
	return msgdata.ConfigInfoJSON{
		ID:           proto.ID,
		DefaultDest:  proto.DefaultDest,
		NodeOperator: proto.NodeOperator,
	}
}
