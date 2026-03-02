package config

import (
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config - унифицированная структура для хранения конфигов.
type Config struct {
	App       AppConfig      `mapstructure:"app"`
	DbConfig  DatabaseConfig `mapstructure:"db"`
	LogConfig LoggerConfig   `mapstructure:"log"`
}

// AppConfig - базовая конфигурация сервера.
type AppConfig struct {
	// Port - порт для запуска сервера.
	Port string `mapstructure:"port"`
	// Debug - флаг для включения дебага.
	Debug bool `mapstructure:"debug"`
	// Env - окружение использования приложения.
	Env string `mapstructure:"env"`
	// Dir - путь до дирректории для хранения отчетов.
	Dir string `mapstructure:"env"`
}

// DatabaseConfig - конфигурация для БД.
type DatabaseConfig struct {
	// Dsn - data source name для подключения к базе данных.
	Dsn string `mapstructure:"dsn"`
}

// LoggerConfig - структура для конфигурации logger.
type LoggerConfig struct {
	// Level уровень логирования.
	Level string `mapstructure:"level"`
}

// Manager - структура для realtime конфигов.
type Manager struct {
	// v - для использования viper.
	v *viper.Viper
	// mu - для конкурентного доступа.
	mu sync.RWMutex
	// cfg - для хранения конфигурации.
	cfg *Config
	// notifyChan - для чистой передачи конфигов.
	notifyChan chan Config
}

// New - создает менеджер и инициализирует канал для управления конфигами.
func New(folder, filename string) (*Manager, error) {
	v := viper.New()
	v.AddConfigPath(folder)
	v.SetConfigName(filename)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// Инициализируем буферизированный канал, так не будет блокировки если полон.
	m := &Manager{
		v:          v,
		notifyChan: make(chan Config, 1),
	}

	// Первичное чтение конфиг файла
	if err := m.reload(); err != nil {
		return nil, err
	}

	// Настройка слежения за изменениями в конфиг файле.
	v.OnConfigChange(func(e fsnotify.Event) {
		if err := m.reload(); err != nil {
			log.Printf("Error reloading config: %v", err)
			return
		}

		// Отправляем новый конфиг в канал.
		select {
		case m.notifyChan <- *m.cfg:
		default:
			// Канал полон (обновления идут слишком часто), пропускаем данное измение.

		}
	})
	v.WatchConfig()

	return m, nil
}

// reload - функция для парсинга конфига в структуру.
func (m *Manager) reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var cfg Config
	if err := m.v.Unmarshal(&cfg); err != nil {
		return err
	}

	m.cfg = &cfg
	return nil
}

// Get - функция, которая возвращает текущий конфиг.
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.cfg
}

// Updates - функция, которая возвращает канал (read-only), c измененными конфигами.
func (m *Manager) Updates() <-chan Config {
	return m.notifyChan
}
