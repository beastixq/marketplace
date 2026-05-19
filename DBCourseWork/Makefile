RPZ_DIR := RPZ
RPZ_TEX := report.tex
PDFLATEX ?= pdflatex

.PHONY: all rpz clean distclean

all: rpz

rpz:
	cd $(RPZ_DIR) && $(PDFLATEX) -interaction=nonstopmode -halt-on-error $(RPZ_TEX)
	cd $(RPZ_DIR) && $(PDFLATEX) -interaction=nonstopmode -halt-on-error $(RPZ_TEX)

clean:
	$(RM) $(RPZ_DIR)/*.aux $(RPZ_DIR)/*.log $(RPZ_DIR)/*.out $(RPZ_DIR)/*.toc
	$(RM) $(RPZ_DIR)/parts/*.aux

distclean: clean
	$(RM) $(RPZ_DIR)/report.pdf
