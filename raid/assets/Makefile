all: index.html index.en.html

%.html: %.md header.html
	pandoc -f markdown -s $< -c ./style.css --metadata "title=Air Raid Alert API (Ukraine, UNOFFICIAL)" -H header.html --highlight-style breezedark -o $@
