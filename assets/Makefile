all: index.html index.en.html

%.html: %.md style.css header.html
	pandoc -f markdown -s $< -c ./style.css --metadata "title=Air Raid Alert API (Ukraine)" -H header.html --self-contained --highlight-style breezedark -o $@
