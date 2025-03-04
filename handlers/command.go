package handlers

import (
	"fmt"
	prErr "proj/errors"
)

// predeclared answers
var (
	Greetings          string = "Приветствую, это система синхронизации goSync. Чтобы узнать доступеные команды, пропишите <help>."
	EnterValidSyncMode string = "Выберете режим синхронизации single/multi. Пример: <sync -s> или <sync --single>. Введите адрес до папки хоста, затем до конечной папки. Пример папки <C:\\Program Files\\mySync>."
	EnterHelpCommand   string = "help - все доступные команды. \n sync -s <folder1> <folder2> - одиночная синхронизация с хостом в первой папке.\n Для подсказки по синхронизации - sync -h.\n"
	TryToCreateFolder  string = "Можно попытаться создать данный каталог, попробовать Y/N?"
)

func CommandHandler(userInput []string) {
	// predeclared commands
	var (
		helpCommand string = "help"
		syncCommand string = "sync"
	)

	switch userInput[0] {
	case helpCommand:
		fmt.Println(EnterHelpCommand)
	case syncCommand:
		if len(userInput)-1 > 0 {
			if userInput[1] == "-h" {
				fmt.Println(EnterValidSyncMode)
			}
		} else {
			fmt.Println(prErr.ErrorCommand)
		}
	default:
		fmt.Println(prErr.ErrorCommand)
	}
}
