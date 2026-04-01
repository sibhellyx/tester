package models

// ContainerInfo - информация о контейнере доступном для chaos-тестирования.
type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
}

// ContainerStats — метрики ресурсов контейнера в момент снимка.
type ContainerStats struct {
	CPUPercent float64 // 0–100 * количество ядер
	MemPercent float64 // 0–100
}
