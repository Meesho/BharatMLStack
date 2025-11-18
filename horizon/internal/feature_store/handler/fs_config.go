package handler

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/Meesho/BharatMLStack/horizon/pkg/zookeeper"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
)

type FsConfigHandler interface {
	GetFsConfigs(ctx *gin.Context) *api.Error
}

type FsConfig struct {
	fsConfigMap map[string][]byte
	zkConfig    *Conf
}

var Mutex sync.Mutex

func NewFsConfigHandlerGenerator() *FsConfig {
	return &FsConfig{
		fsConfigMap: make(map[string][]byte),
		zkConfig:    &Conf{},
	}
}
func (f *FsConfig) GetFsConfigs(ctx *gin.Context) *api.Error {
	data, ok := f.fsConfigMap["fs_config"]
	if !ok {
		return api.NewInternalServerError("Failed to retrieve configs")
	}
	response := &FsConfigResponse{
		Data: data,
	}
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create zstd decoder")

	}
	decompressedBytes, err := decoder.DecodeAll(data, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decompress bytes")

	}

	zkconfig1 := Conf{}
	// Unmarshal the decompressed bytes into zkConfig
	err = json.Unmarshal(decompressedBytes, &zkconfig1)
	ctx.JSON(200, response)
	return nil
}

func (f *FsConfig) InitFsConfig(config configs.Configs) {
	zookeeper.InitZKConnection(config)
	zookeeper.WatchZkNode()
	go f.ListenZk()
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create zstd decoder")
		return
	}
	decompressedBytes, err := decoder.DecodeAll(f.fsConfigMap["fs_config"], nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decompress bytes")
		return
	}

	zkconfig1 := Conf{}
	// Unmarshal the decompressed bytes into zkConfig
	err = json.Unmarshal(decompressedBytes, &zkconfig1)

	log.Info().Msg("Initializing FS Config Handler")
}

func (f *FsConfig) ListenZk() {
	for {
		select {
		case data := <-zookeeper.ZKChannel:
			var localZkConf *Conf
			var compressedBytes []byte
			err := json.Unmarshal(data, &localZkConf)
			if err != nil {
				log.Error().Err(err).Msg("Failed to unmarshal JSON")
			} else {
				Mutex.Lock()
				f.zkConfig = localZkConf

				// Convert ZkConf to byte array
				zkConfBytes, err := json.Marshal(f.zkConfig)
				if err != nil {
					log.Error().Err(err).Msg("Failed to marshal ZkConf")
				} else {
					// Print size before compression
					fmt.Printf("Size before compression: %d bytes\n", len(zkConfBytes))

					// Apply zstd compression
					encoder, err := zstd.NewWriter(nil)
					if err != nil {
						log.Error().Err(err).Msg("Failed to create zstd encoder")
					}
					compressedBytes = encoder.EncodeAll(zkConfBytes, nil)

					// Print size after compression
					fmt.Printf("Size after compression: %d bytes\n", len(compressedBytes))
				}
				Mutex.Unlock()

			}

			if len(compressedBytes) != 0 {
				f.fsConfigMap["fs_config"] = compressedBytes
			}
		}
	}
}
