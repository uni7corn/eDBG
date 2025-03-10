package utils

import (
	"log"
	"context"
)

type GlobalConfig struct {
	initialOffsets []uint64
	packageName string
	libName string
	logger *log.Logger
}

func NewGlobalConfig() *GlobalConfig {
    return &GlobalConfig{}
}