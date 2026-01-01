//go:build linux || windows

package service

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kardianos/service"
	"github.com/weii/actime/internal/config"
)

var (
	svc service.Service
)

type program struct {
	svc *Service
}

func (p *program) Start(s service.Service) error {
	log := service.ConsoleLogger
	log.Info("Starting Actime service...")

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create service
	p.svc, err = NewService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Start service in background
	go func() {
		if err := p.svc.Start(); err != nil {
			log.Error("Failed to start service", "error", err)
		}
	}()

	return nil
}

func (p *program) Stop(s service.Service) error {
	log := service.ConsoleLogger
	log.Info("Stopping Actime service...")

	if p.svc != nil {
		if err := p.svc.Stop(); err != nil {
			log.Error("Failed to stop service", "error", err)
			return err
		}
	}

	return nil
}

// InstallService installs the Actime service
func InstallService() error {
	cfg := &service.Config{
		Name:        "Actime",
		DisplayName: "Actime Time Tracker",
		Description: "Tracks application usage time",
	}

	prg := &program{}
	s, err := service.New(prg, cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("Service installed successfully")
	return nil
}

// UninstallService uninstalls the Actime service
func UninstallService() error {
	cfg := &service.Config{
		Name:        "Actime",
		DisplayName: "Actime Time Tracker",
		Description: "Tracks application usage time",
	}

	prg := &program{}
	s, err := service.New(prg, cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	fmt.Println("Service uninstalled successfully")
	return nil
}

// RunService runs the Actime as a system service
func RunService() error {
	cfg := &service.Config{
		Name:        "Actime",
		DisplayName: "Actime Time Tracker",
		Description: "Tracks application usage time",
	}

	prg := &program{}
	s, err := service.New(prg, cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	logger, err := s.Logger(nil)
	if err != nil {
		return fmt.Errorf("failed to get service logger: %w", err)
	}

	err = s.Run()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

// RunForeground runs the Actime in foreground mode
func RunForeground() error {
	fmt.Println("Running Actime in foreground mode...")
	fmt.Println("Press Ctrl+C to stop")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create service
	svc, err := NewService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Start service
	if err := svc.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nReceived shutdown signal")

	// Stop service
	if err := svc.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("Actime stopped")
	return nil
}