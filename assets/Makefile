all: index.html index.en.html

%.html: %.md
	pandoc -f markdown -s $< -c ./style.css --metadata "title=Air Raid Alert API (Ukraine)" --self-contained --highlight-style breezedark -o $@
