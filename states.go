package main

import "time"

type State struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Alert   bool       `json:"alert"`
	Changed *time.Time `json:"changed"`
}

var States = []State{
	{1, "Вінницька область", false, nil},
	{2, "Волинська область", false, nil},
	{3, "Дніпропетровська область", false, nil},
	{4, "Донецька область", false, nil},
	{5, "Житомирська область", false, nil},
	{6, "Закарпатська область", false, nil},
	{7, "Запорізька область", false, nil},
	{8, "Івано-Франківська область", false, nil},
	{9, "Київська область", false, nil},
	{10, "Кіровоградська область", false, nil},
	{11, "Луганська область", false, nil},
	{12, "Львівська область", false, nil},
	{13, "Миколаївська область", false, nil},
	{14, "Одеська область", false, nil},
	{15, "Полтавська область", false, nil},
	{16, "Рівненська область", false, nil},
	{17, "Сумська область", false, nil},
	{18, "Тернопільська область", false, nil},
	{19, "Харківська область", false, nil},
	{20, "Херсонська область", false, nil},
	{21, "Хмельницька область", false, nil},
	{22, "Черкаська область", false, nil},
	{23, "Чернівецька область", false, nil},
	{24, "Чернігівська область", false, nil},
	{25, "м. Київ", false, nil},
}
