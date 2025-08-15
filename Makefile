CHART_ROOT = deploy/charts/ghes-schedule-scanner

# Generate documentation for all charts
.PHONY: docs
docs:
	@echo "Generating docs for $(CHART_ROOT) using README.md.gotmpl"; \
	helm-docs --chart-search-root "$(CHART_ROOT)" --template-files="$(CHART_ROOT)/ci/README.md.gotmpl" --sort-values-order file --log-level info
