PLANTUML = java -DPLANTUML_LIMIT_SIZE=16384 -jar plantuml.jar
SRC_DIR  := diagrams/src
OUT_DIR  := diagrams/out

PUML_FILES := $(wildcard $(SRC_DIR)/*.puml)
PNG_FILES  := $(patsubst $(SRC_DIR)/%.puml,$(OUT_DIR)/%.png,$(PUML_FILES))
SVG_FILES  := $(patsubst $(SRC_DIR)/%.puml,$(OUT_DIR)/%.svg,$(PUML_FILES))

.PHONY: all png svg clean

all: png

png: $(OUT_DIR) $(PNG_FILES)

svg: $(OUT_DIR) $(SVG_FILES)

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

$(OUT_DIR)/%.png: $(SRC_DIR)/%.puml | $(OUT_DIR)
	$(PLANTUML) -tpng -o $(abspath $(OUT_DIR)) $<

$(OUT_DIR)/%.svg: $(SRC_DIR)/%.puml | $(OUT_DIR)
	$(PLANTUML) -tsvg -o $(abspath $(OUT_DIR)) $<

clean:
	rm -f $(OUT_DIR)/*.png $(OUT_DIR)/*.svg
