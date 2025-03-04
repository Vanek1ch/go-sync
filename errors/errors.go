package errors

// predeclared errors
var (
	ErrorCommand         string = "Такой команды не существует."
	ErrorFolder          string = "Ошибка! Вы ввели неверный путь или он не сущетсвует."
	ErrorAnswer          string = "Ошибка при получении ответа от пользователя."
	ErrorMkdir           string = "Произошла ошибка при попытке создать каталог."
	ErrorSyncTypeMissing        = "Ошибка, данный метод синхронизации находится в разработке."
)
