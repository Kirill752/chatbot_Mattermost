package main

import (
	"log"
	storge "vk_tarantool/Internal/Storge"
	"vk_tarantool/Internal/handlers/pool"
)

func main() {
	// TODO: Загрузка конфига
	// TODO: Инициализация mattermost
	// app := application.New("./Config/conf.yaml")
	// app.SetupGracefulShutdown()
	// app.ListenToEvents()
	// TODO: Инициализация tarantool
	strg, err := storge.New("127.0.0.1:3301", "storage", "passw0rd")
	if err != nil {
		log.Fatal(err)
	}
	strg.Save(&pool.Pool{
		ID:       2,
		Title:    "Ваш любимый язык программирования?",
		Variants: []string{"Go", "C++", "Lua"},
	})
	// err = strg.Delete(2)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// d, err := strg.Select(2)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(d)
	// strg.SelectSome(0, 100)
	// getAllPolls(conn)
	// TODO: Инициализация обработчика команд
	// TODO: Запуск прослушивания событий
	// // msg := `опРос 1 "ваш любимый фрукт?" яблоко груша киви мандарин апельсин`
	// // pool, err := pool.CreatePool(msg)
	// // if err != nil {
	// // 	log.Fatal(err)
	// // }
	// // fmt.Println(pool)
}
