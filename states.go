package main

import "time"

type State struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	NameEn  string     `json:"name_en"`
	Alert   bool       `json:"alert"`
	Changed *time.Time `json:"changed"`
}

var States = []State{
	{1, "Вінницька область", "Vinnytsia oblast", false, nil},
	{2, "Волинська область", "Volyn oblast", false, nil},
	{3, "Дніпропетровська область", "Dnipropetrovsk oblast", false, nil},
	{4, "Донецька область", "Donetsk oblast", false, nil},
	{5, "Житомирська область", "Zhytomyr oblast", false, nil},
	{6, "Закарпатська область", "Zakarpattia oblast", false, nil},
	{7, "Запорізька область", "Zaporizhzhia oblast", false, nil},
	{8, "Івано-Франківська область", "Ivano-Frankivsk oblast", false, nil},
	{9, "Київська область", "Kyiv oblast", false, nil},
	{10, "Кіровоградська область", "Kirovohrad oblast", false, nil},
	{11, "Луганська область", "Luhansk oblast", false, nil},
	{12, "Львівська область", "Lviv oblast", false, nil},
	{13, "Миколаївська область", "Mykolaiv oblast", false, nil},
	{14, "Одеська область", "Odesa oblast", false, nil},
	{15, "Полтавська область", "Poltava oblast", false, nil},
	{16, "Рівненська область", "Rivne oblast", false, nil},
	{17, "Сумська область", "Sumy oblast", false, nil},
	{18, "Тернопільська область", "Ternopil oblast", false, nil},
	{19, "Харківська область", "Kharkiv oblast", false, nil},
	{20, "Херсонська область", "Kherson oblast", false, nil},
	{21, "Хмельницька область", "Khmelnytskyi oblast", false, nil},
	{22, "Черкаська область", "Cherkasy oblast", false, nil},
	{23, "Чернівецька область", "Chernivtsi oblast", false, nil},
	{24, "Чернігівська область", "Chernihiv oblast", false, nil},
	{25, "м. Київ", "Kyiv", false, nil},
}
