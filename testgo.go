package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	EV_KEY = 0x01
	EV_REL = 0x02
	EV_ABS = 0x03
)

// Input event structure
type InputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// Mouse button codes
var buttonNames = map[uint16]string{
	0x110: "BTN_LEFT (Левая кнопка)",
	0x111: "BTN_RIGHT (Правая кнопка)",
	0x112: "BTN_MIDDLE (Средняя кнопка)",
	0x113: "BTN_SIDE (Боковая кнопка)",
	0x114: "BTN_EXTRA (Доп. кнопка)",
	0x115: "BTN_FORWARD (Вперёд)",
	0x116: "BTN_BACK (Назад)",
	0x117: "BTN_TASK (Task)",
	0x118: "BTN_JOYSTICK",
	0x119: "BTN_THUMB",
	0x11a: "BTN_THUMB2",
	0x11b: "BTN_TOP",
	0x11c: "BTN_TOP2",
	0x11d: "BTN_PINKIE",
	0x11e: "BTN_BASE",
	0x11f: "BTN_BASE2",
}

func main() {
	// Ищем устройства мыши в /dev/input
	devices, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		log.Fatalf("Ошибка поиска устройств: %v", err)
	}

	fmt.Println("=== Тест событий мыши ===")
	fmt.Println("Поиск устройств мыши...")
	fmt.Println()

	var mouseDevices []string

	// Проверяем каждое устройство
	for _, device := range devices {
		// Пытаемся прочитать имя устройства
		if isMouse(device) {
			mouseDevices = append(mouseDevices, device)
			fmt.Printf("✓ Найдено: %s\n", device)
		}
	}

	if len(mouseDevices) == 0 {
		log.Fatal("Устройства мыши не найдены. Запустите программу с sudo.")
	}

	fmt.Println()
	fmt.Println("Запуск мониторинга событий...")
	fmt.Println("Нажмите любую кнопку мыши для определения её кода.")
	fmt.Println("Нажмите Ctrl+C для выхода.")
	fmt.Println()

	// Открываем все найденные устройства мыши
	for _, device := range mouseDevices {
		go monitorDevice(device)
	}

	// Блокируем основной поток
	select {}
}

func isMouse(devicePath string) bool {
	// Пробуем открыть устройство для чтения
	file, err := os.Open(devicePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Получаем имя устройства через ioctl
	var name [256]byte
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(0x80ff4506), // EVIOCGNAME
		uintptr(unsafe.Pointer(&name)),
	)

	if errno != 0 {
		return false
	}

	deviceName := string(name[:])
	deviceName = strings.ToLower(deviceName)

	// Проверяем, что это мышь
	return strings.Contains(deviceName, "mouse") ||
		strings.Contains(deviceName, "touchpad") ||
		strings.Contains(deviceName, "pointer")
}

func monitorDevice(devicePath string) {
	file, err := os.Open(devicePath)
	if err != nil {
		log.Printf("Не удалось открыть %s: %v (нужны права root)\n", devicePath, err)
		return
	}
	defer file.Close()

	// Получаем имя устройства
	var name [256]byte
	syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(0x80ff4506),
		uintptr(unsafe.Pointer(&name)),
	)

	deviceName := strings.TrimRight(string(name[:]), "\x00")

	for {
		var event InputEvent
		err := binary.Read(file, binary.LittleEndian, &event)
		if err != nil {
			log.Printf("Ошибка чтения из %s: %v\n", devicePath, err)
			return
		}

		// Обрабатываем события кнопок (EV_KEY)
		if event.Type == EV_KEY {
			action := "???"
			if event.Value == 1 {
				action = "НАЖАТА"
			} else if event.Value == 0 {
				action = "ОТПУЩЕНА"
			} else if event.Value == 2 {
				action = "УДЕРЖАНИЕ"
			}

			buttonName := buttonNames[event.Code]
			if buttonName == "" {
				buttonName = fmt.Sprintf("Неизвестная кнопка 0x%03X", event.Code)
			}

			fmt.Printf("🖱️  %s: %s | Код: 0x%03X (дес: %d) | Устройство: %s\n",
				action, buttonName, event.Code, event.Code, deviceName)
		}

		// Показываем также события движения (для отладки)
		if event.Type == EV_REL {
			// Не выводим каждое событие движения, чтобы не захламлять вывод
			// fmt.Printf("  [движение: код=0x%03X значение=%d]\n", event.Code, event.Value)
		}
	}
}
