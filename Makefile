CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION := $(subst v,,$(CURRENT_TAG))
MAJOR := $(word 1,$(subst ., ,$(VERSION)))
MINOR := $(word 2,$(subst ., ,$(VERSION)))
PATCH := $(word 3,$(subst ., ,$(VERSION)))

.PHONY: release-patch release-minor release-major

# 升级补丁版本 x.y.Z
release-patch:
	$(eval NEW_PATCH := $(shell echo $$(($(PATCH)+1))))
	$(eval NEW_TAG := v$(MAJOR).$(MINOR).$(NEW_PATCH))
	@echo "$(CURRENT_TAG) → $(NEW_TAG)"
	git tag $(NEW_TAG)
	git push origin $(NEW_TAG)

# 升级次版本 x.Y.0
release-minor:
	$(eval NEW_MINOR := $(shell echo $$(($(MINOR)+1))))
	$(eval NEW_TAG := v$(MAJOR).$(NEW_MINOR).0)
	@echo "$(CURRENT_TAG) → $(NEW_TAG)"
	git tag $(NEW_TAG)
	git push origin $(NEW_TAG)

# 升级主版本 X.0.0
release-major:
	$(eval NEW_MAJOR := $(shell echo $$(($(MAJOR)+1))))
	$(eval NEW_TAG := v$(NEW_MAJOR).0.0)
	@echo "$(CURRENT_TAG) → $(NEW_TAG)"
	git tag $(NEW_TAG)
	git push origin $(NEW_TAG)
