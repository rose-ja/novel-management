package imports

import (
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"novel-management/backend/internal/model"
)

// NormalizeImagePath 规范化 Markdown 中的相对图片路径，拒绝绝对路径与越界。
func NormalizeImagePath(raw string) (string, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, `\`, "/"))
	value = strings.TrimPrefix(value, "./")
	if value == "" {
		return "", fmt.Errorf("图片路径不能为空")
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, "://") || (len(value) >= 2 && value[1] == ':') {
		return "", fmt.Errorf("图片路径必须是相对路径")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("图片路径越界")
	}
	return clean, nil
}

// MatchImages 对照 Markdown 引用与 zip 清单，返回匹配摘要与错误。
func MatchImages(spec *Spec, inventory *Inventory) (model.ImportImageSummary, []Issue) {
	var summary model.ImportImageSummary
	summary.Missing = []string{}
	summary.Extra = []string{}
	summary.Unsupported = []string{}
	summary.TooLarge = []string{}
	var issues []Issue

	referenced := map[string]bool{}
	for _, character := range spec.Characters {
		if strings.TrimSpace(character.Image) == "" {
			continue
		}
		normalized, err := NormalizeImagePath(character.Image)
		if err != nil {
			issues = append(issues, Issue{Line: character.Line, Field: "人物/" + character.Name + "/图片", Code: CodePathTraversal, Message: err.Error()})
			continue
		}
		referenced[normalized] = true
	}

	files := map[string]ImageFile{}
	if inventory != nil {
		files = inventory.Files
	}
	for name := range referenced {
		file, ok := files[name]
		if !ok {
			summary.Missing = append(summary.Missing, name)
			issues = append(issues, Issue{Line: 0, Field: "图片/" + name, Code: CodeImageMissing, Message: fmt.Sprintf("图片 %s 在 zip 中缺失", name)})
			continue
		}
		if file.Size > model.MaxImageBytes {
			summary.TooLarge = append(summary.TooLarge, name)
			issues = append(issues, Issue{Line: 0, Field: "图片/" + name, Code: CodeImageTooLarge, Message: fmt.Sprintf("图片 %s 超过 10MB", name)})
			continue
		}
		if !supportedMIME(file.MIME) {
			summary.Unsupported = append(summary.Unsupported, name)
			issues = append(issues, Issue{Line: 0, Field: "图片/" + name, Code: CodeImageUnsupported, Message: fmt.Sprintf("图片 %s 仅支持 JPG、PNG 或 WebP", name)})
			continue
		}
		summary.Matched++
	}
	for name, file := range files {
		if referenced[name] || !looksLikeImage(name, file.MIME) {
			continue
		}
		summary.Extra = append(summary.Extra, name)
		issues = append(issues, Issue{Line: 0, Field: "图片/" + name, Code: CodeImageExtra, Message: fmt.Sprintf("图片 %s 未被 Markdown 引用", name)})
	}
	sort.Strings(summary.Missing)
	sort.Strings(summary.Extra)
	sort.Strings(summary.Unsupported)
	sort.Strings(summary.TooLarge)
	return summary, issues
}

func supportedMIME(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func looksLikeImage(name, mime string) bool {
	if strings.HasPrefix(mime, "image/") {
		return true
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".svg", ".tif", ".tiff", ".avif":
		return true
	default:
		return false
	}
}

// DetectImageMIME 通过文件头识别图片类型，供 zip 清单使用。
func DetectImageMIME(header []byte) string {
	switch {
	case len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF:
		return "image/jpeg"
	case len(header) >= 8 && string(header[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(header) >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP":
		return "image/webp"
	default:
		detected := http.DetectContentType(header)
		if strings.HasPrefix(detected, "image/") {
			return detected
		}
		return ""
	}
}

// Validate 校验语义规则并返回计数、图片摘要与错误/警告。
func Validate(spec *Spec, inventory *Inventory) ValidationResult {
	result := ValidationResult{
		Errors:   []Issue{},
		Warnings: []Issue{},
	}
	categories := CategoryPlan(spec)
	if len(categories) > model.MaxSystemCategories {
		result.Errors = append(result.Errors, Issue{Line: lineOfCategories(spec), Field: "体系类别", Code: CodeCategoryLimit, Message: fmt.Sprintf("体系类别最多 %d 个", model.MaxSystemCategories)})
	}

	categoryNames := make(map[string]bool, len(categories))
	for _, category := range categories {
		categoryNames[category.Name] = true
	}

	nodeSet := validateNodes(spec, categoryNames, &result)
	characterNames := collectNames(spec, sectionCharacters, &result)
	factionNames := collectNames(spec, sectionFactions, &result)
	locationNames := collectNames(spec, sectionLocations, &result)
	collectNames(spec, sectionRankings, &result)

	validateWork(spec, &result)
	validateFactions(spec, factionNames, characterNames, &result)
	validateLocations(spec, locationNames, factionNames, &result)
	validateCharacters(spec, categoryNames, characterNames, factionNames, locationNames, nodeSet, &result)
	validateRankings(spec, characterNames, &result)
	validateChronicles(spec, characterNames, factionNames, &result)

	images, imageIssues := MatchImages(spec, inventory)
	result.Images = images
	result.Errors = append(result.Errors, imageIssues...)

	result.Counts = model.ImportCounts{
		Characters:       len(spec.Characters),
		Factions:         len(spec.Factions),
		Locations:        len(spec.Locations),
		Rankings:         len(spec.Rankings),
		SystemCategories: len(categories),
		SystemNodes:      len(spec.Nodes),
		Chronicles:       len(spec.Chronicles),
		Images:           images.Matched,
	}
	result.Counts.Reviewable = result.Counts.Characters + result.Counts.Factions +
		result.Counts.Locations + result.Counts.Rankings + result.Counts.SystemNodes
	return result
}

// CategoryPlan 返回含内置境界的类别列表；境界始终存在且唯一。
func CategoryPlan(spec *Spec) []CategorySpec {
	var realm *CategorySpec
	seen := map[string]bool{}
	var result []CategorySpec
	for _, category := range spec.Categories {
		name := strings.TrimSpace(category.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if name == model.BuiltinRealmCategory {
			realm = &CategorySpec{Name: name, Builtin: true, Line: category.Line}
			result = append(result, *realm)
			continue
		}
		if category.Builtin {
			continue
		}
		result = append(result, CategorySpec{Name: name, Line: category.Line})
	}
	if realm == nil {
		result = append([]CategorySpec{{Name: model.BuiltinRealmCategory, Builtin: true, Line: 0}}, result...)
	}
	return result
}

func lineOfCategories(spec *Spec) int {
	if len(spec.Categories) > 0 {
		return spec.Categories[0].Line
	}
	return 1
}

func validateWork(spec *Spec, result *ValidationResult) {
	if _, _, err := model.NormalizeName(spec.Work.Name); err != nil {
		result.Errors = append(result.Errors, Issue{Line: spec.Work.Line, Field: "作品信息/名称", Code: CodeFieldRequired, Message: err.Error()})
	}
	if _, err := model.NormalizeDescription(&spec.Work.Description); err != nil {
		result.Errors = append(result.Errors, Issue{Line: spec.Work.Line, Field: "作品信息/简介", Code: CodeInvalid, Message: err.Error()})
	}
	if len(spec.Categories) == 0 {
		result.Errors = append(result.Errors, Issue{Line: 1, Field: "体系类别", Code: CodeNoCategories, Message: "模板必须声明体系类别（至少包含境界）"})
	}
	seenCategories := map[string]bool{}
	for _, category := range spec.Categories {
		name := strings.TrimSpace(category.Name)
		if seenCategories[name] {
			result.Errors = append(result.Errors, Issue{Line: category.Line, Field: "体系类别/" + name, Code: CodeDuplicate, Message: fmt.Sprintf("类别「%s」重复", name)})
			continue
		}
		seenCategories[name] = true
		if name == model.BuiltinRealmCategory && !category.Builtin {
			continue
		}
		if category.Builtin && name != model.BuiltinRealmCategory {
			result.Errors = append(result.Errors, Issue{Line: category.Line, Field: "体系类别/" + name, Code: CodeInvalid, Message: "仅「境界」可标记为内置类别"})
			continue
		}
		if _, _, err := model.NormalizeKey(name, "name"); err != nil {
			result.Errors = append(result.Errors, Issue{Line: category.Line, Field: "体系类别/" + name, Code: CodeInvalid, Message: err.Error()})
		}
	}
}

func validateNodes(spec *Spec, categories map[string]bool, result *ValidationResult) map[string]bool {
	declared := map[string]bool{} // category\x00name：所有已声明节点，用于父级查找
	topLevel := map[string]bool{} // category\x00name：仅一级节点可作为父级
	nodes := map[string]bool{}    // 供人物引用校验：一级用 cat\x00name，二级用 cat\x00parent\x00name
	seen := map[string]bool{}
	for _, node := range spec.Nodes {
		if !categories[node.Category] {
			result.Errors = append(result.Errors, Issue{Line: node.Line, Field: "体系节点/" + node.Name, Code: CodeReferenceMissing, Message: fmt.Sprintf("类别「%s」未定义", node.Category)})
			continue
		}
		if _, _, err := model.NormalizeKey(node.Name, "name"); err != nil {
			result.Errors = append(result.Errors, Issue{Line: node.Line, Field: "体系节点/" + node.Name, Code: CodeInvalid, Message: err.Error()})
			continue
		}
		key := node.Category + "\x00" + node.Parent + "\x00" + node.Name
		if seen[key] {
			result.Errors = append(result.Errors, Issue{Line: node.Line, Field: "体系节点/" + node.Name, Code: CodeDuplicate, Message: fmt.Sprintf("同一父级下节点「%s」重复", node.Name)})
			continue
		}
		seen[key] = true

		nameKey := node.Category + "\x00" + node.Name
		if node.Parent != "" {
			parentKey := node.Category + "\x00" + node.Parent
			if !declared[parentKey] {
				result.Errors = append(result.Errors, Issue{Line: node.Line, Field: "体系节点/" + node.Name, Code: CodeNodeParent, Message: fmt.Sprintf("上级节点「%s」不存在或晚于子节点声明", node.Parent)})
				continue
			}
			if !topLevel[parentKey] {
				result.Errors = append(result.Errors, Issue{Line: node.Line, Field: "体系节点/" + node.Name, Code: CodeNodeLevel, Message: "体系节点最多两级"})
				continue
			}
			nodes[key] = true
			declared[nameKey] = true
			continue
		}
		topLevel[nameKey] = true
		declared[nameKey] = true
		nodes[nameKey] = true
	}
	return nodes
}

func collectNames(spec *Spec, field string, result *ValidationResult) map[string]bool {
	names := map[string]bool{}
	switch field {
	case sectionCharacters:
		for _, value := range spec.Characters {
			registerName(value.Name, value.Line, field, names, result)
		}
	case sectionFactions:
		for _, value := range spec.Factions {
			registerName(value.Name, value.Line, field, names, result)
		}
	case sectionLocations:
		for _, value := range spec.Locations {
			registerName(value.Name, value.Line, field, names, result)
		}
	case sectionRankings:
		for _, value := range spec.Rankings {
			registerName(value.Name, value.Line, field, names, result)
		}
	}
	return names
}

func registerName(raw string, line int, field string, names map[string]bool, result *ValidationResult) {
	value := strings.TrimSpace(raw)
	if value == "" {
		result.Errors = append(result.Errors, Issue{Line: line, Field: field + "/名称", Code: CodeFieldRequired, Message: "名称不能为空"})
		return
	}
	key := strings.ToLower(value)
	if names[key] {
		result.Errors = append(result.Errors, Issue{Line: line, Field: field + "/" + value, Code: CodeDuplicate, Message: fmt.Sprintf("名称「%s」重复", value)})
		return
	}
	names[key] = true
}

func validateFactions(spec *Spec, factions, characters map[string]bool, result *ValidationResult) {
	for _, faction := range spec.Factions {
		if _, _, err := model.NormalizeFactionName(faction.Name); err != nil {
			result.Errors = append(result.Errors, Issue{Line: faction.Line, Field: "势力/" + faction.Name, Code: CodeInvalid, Message: err.Error()})
		}
		if _, err := model.NormalizeFactionDescription(&faction.Description); err != nil {
			result.Errors = append(result.Errors, Issue{Line: faction.Line, Field: "势力/" + faction.Name + "/简介", Code: CodeInvalid, Message: err.Error()})
		}
		if faction.Parent != "" && !factions[strings.ToLower(faction.Parent)] {
			result.Errors = append(result.Errors, Issue{Line: faction.Line, Field: "势力/" + faction.Name + "/上级", Code: CodeReferenceMissing, Message: fmt.Sprintf("上级势力「%s」不存在", faction.Parent)})
		}
		if faction.Leader != "" && !characters[strings.ToLower(faction.Leader)] {
			result.Errors = append(result.Errors, Issue{Line: faction.Line, Field: "势力/" + faction.Name + "/首领", Code: CodeReferenceMissing, Message: fmt.Sprintf("首领人物「%s」不存在", faction.Leader)})
		}
	}
	parents := map[string]string{}
	for _, faction := range spec.Factions {
		parents[strings.ToLower(faction.Name)] = strings.ToLower(faction.Parent)
	}
	for _, faction := range spec.Factions {
		seen := map[string]bool{}
		current := strings.ToLower(faction.Name)
		for current != "" {
			if seen[current] {
				result.Errors = append(result.Errors, Issue{Line: faction.Line, Field: "势力/" + faction.Name + "/上级", Code: CodeInvalid, Message: "势力上级形成环"})
				break
			}
			seen[current] = true
			current = parents[current]
		}
	}
}

func validateLocations(spec *Spec, locations, factions map[string]bool, result *ValidationResult) {
	for _, location := range spec.Locations {
		if _, _, err := model.NormalizeLocationName(location.Name); err != nil {
			result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name, Code: CodeInvalid, Message: err.Error()})
		}
		if _, err := model.NormalizeLocationType(&location.TypeText); err != nil {
			result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name + "/类型", Code: CodeInvalid, Message: err.Error()})
		}
		if _, err := model.NormalizeLocationDescription(&location.Description); err != nil {
			result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name + "/简介", Code: CodeInvalid, Message: err.Error()})
		}
		if location.Parent != "" && !locations[strings.ToLower(location.Parent)] {
			result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name + "/上级", Code: CodeReferenceMissing, Message: fmt.Sprintf("上级地点「%s」不存在", location.Parent)})
		}
		for _, relation := range location.Factions {
			if !factions[strings.ToLower(relation.Name)] {
				result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name + "/势力", Code: CodeReferenceMissing, Message: fmt.Sprintf("势力「%s」不存在", relation.Name)})
			}
			if _, err := model.NormalizeRelationNote(&relation.Note); err != nil {
				result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name + "/势力备注", Code: CodeInvalid, Message: err.Error()})
			}
		}
	}
	parents := map[string]string{}
	for _, location := range spec.Locations {
		parents[strings.ToLower(location.Name)] = strings.ToLower(location.Parent)
	}
	for _, location := range spec.Locations {
		seen := map[string]bool{}
		current := strings.ToLower(location.Name)
		for current != "" {
			if seen[current] {
				result.Errors = append(result.Errors, Issue{Line: location.Line, Field: "地点/" + location.Name + "/上级", Code: CodeInvalid, Message: "地点上级形成环"})
				break
			}
			seen[current] = true
			current = parents[current]
		}
	}
}

func validateCharacters(spec *Spec, categories, characters, factions, locations, nodes map[string]bool, result *ValidationResult) {
	for _, character := range spec.Characters {
		if _, _, err := model.NormalizeCharacterName(character.Name); err != nil {
			result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name, Code: CodeInvalid, Message: err.Error()})
		}
		if _, err := model.NormalizeCharacterDescription(&character.Description); err != nil {
			result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/简介", Code: CodeInvalid, Message: err.Error()})
		}
		validateNameValues(character, character.Aliases, "别名", result)
		validateNameValues(character, character.Titles, "称号", result)
		if character.Faction != "" && !factions[strings.ToLower(character.Faction)] {
			result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/势力", Code: CodeReferenceMissing, Message: fmt.Sprintf("势力「%s」不存在", character.Faction)})
		}
		for _, relation := range character.Locations {
			if !locations[strings.ToLower(relation.Name)] {
				result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/地点", Code: CodeReferenceMissing, Message: fmt.Sprintf("地点「%s」不存在", relation.Name)})
			}
			if _, err := model.NormalizeRelationNote(&relation.Note); err != nil {
				result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/地点备注", Code: CodeInvalid, Message: err.Error()})
			}
		}
		for category, nodePath := range character.Extras {
			if !categories[category] {
				switch category {
				case "类型", "关系":
					result.Warnings = append(result.Warnings, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + category, Code: CodeInvalid, Message: "类型文本仅记录，不建立引用"})
				default:
					result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + category, Code: CodeUnknownField, Message: fmt.Sprintf("未知字段「%s」", category)})
				}
				continue
			}
			parts := strings.Split(nodePath, ">")
			if len(parts) > 2 || strings.TrimSpace(parts[0]) == "" {
				result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + category, Code: CodeNodeLevel, Message: "体系引用最多两级"})
				continue
			}
			parent := ""
			name := strings.TrimSpace(parts[0])
			if len(parts) == 2 {
				parent = strings.TrimSpace(parts[0])
				name = strings.TrimSpace(parts[1])
			}
			if !nodeExists(nodes, category, parent, name) {
				result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + category, Code: CodeReferenceMissing, Message: fmt.Sprintf("体系节点「%s」不存在", nodePath)})
			}
		}
	}
}

func nodeExists(nodes map[string]bool, category, parent, name string) bool {
	if parent == "" {
		return nodes[category+"\x00"+name]
	}
	return nodes[category+"\x00"+parent+"\x00"+name]
}

func validateNameValues(character CharacterSpec, values []string, field string, result *ValidationResult) {
	if len(values) > model.MaxCharacterNames {
		result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + field, Code: CodeInvalid, Message: fmt.Sprintf("%s最多 %d 个", field, model.MaxCharacterNames)})
	}
	seen := map[string]bool{}
	for _, value := range values {
		_, key, err := model.NormalizeCharacterNameValue(value, field)
		if err != nil {
			result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + field, Code: CodeInvalid, Message: err.Error()})
			continue
		}
		if seen[key] {
			result.Errors = append(result.Errors, Issue{Line: character.Line, Field: "人物/" + character.Name + "/" + field, Code: CodeDuplicate, Message: fmt.Sprintf("%s「%s」重复", field, value)})
		}
		seen[key] = true
	}
}

func validateRankings(spec *Spec, characters map[string]bool, result *ValidationResult) {
	for _, ranking := range spec.Rankings {
		if _, _, err := model.NormalizeRankingName(ranking.Name); err != nil {
			result.Errors = append(result.Errors, Issue{Line: ranking.Line, Field: "榜单/" + ranking.Name, Code: CodeInvalid, Message: err.Error()})
		}
		if ranking.SeatCount == 0 {
			ranking.SeatCount = maxSeat(ranking.Seats)
		}
		if err := model.NormalizeSeatCount(ranking.SeatCount); err != nil {
			result.Errors = append(result.Errors, Issue{Line: ranking.Line, Field: "榜单/" + ranking.Name + "/席位数", Code: CodeInvalid, Message: err.Error()})
		}
		seats := map[int]bool{}
		occupants := map[string]bool{}
		for _, seat := range ranking.Seats {
			if seat.No < 1 || seat.No > ranking.SeatCount {
				result.Errors = append(result.Errors, Issue{Line: seat.Line, Field: "榜单/" + ranking.Name + "/席位", Code: CodeInvalid, Message: fmt.Sprintf("席位号 %d 超出范围", seat.No)})
			}
			if seats[seat.No] {
				result.Errors = append(result.Errors, Issue{Line: seat.Line, Field: "榜单/" + ranking.Name + "/席位", Code: CodeDuplicate, Message: fmt.Sprintf("席位号 %d 重复", seat.No)})
			}
			seats[seat.No] = true
			if seat.Character != "" {
				key := strings.ToLower(seat.Character)
				if !characters[key] {
					result.Errors = append(result.Errors, Issue{Line: seat.Line, Field: "榜单/" + ranking.Name + "/" + fmt.Sprintf("第%d席", seat.No), Code: CodeReferenceMissing, Message: fmt.Sprintf("人物「%s」不存在", seat.Character)})
				}
				if occupants[key] {
					result.Errors = append(result.Errors, Issue{Line: seat.Line, Field: "榜单/" + ranking.Name + "/席位", Code: CodeDuplicate, Message: fmt.Sprintf("人物「%s」重复占席", seat.Character)})
				}
				occupants[key] = true
			}
			if _, err := model.NormalizeSeatNote(&seat.Note); err != nil {
				result.Errors = append(result.Errors, Issue{Line: seat.Line, Field: "榜单/" + ranking.Name + "/席位备注", Code: CodeInvalid, Message: err.Error()})
			}
		}
	}
}

func maxSeat(seats []SeatSpec) int {
	max := 0
	for _, seat := range seats {
		if seat.No > max {
			max = seat.No
		}
	}
	if max == 0 {
		max = model.DefaultRankingSeats
	}
	return max
}

func validateChronicles(spec *Spec, characters, factions map[string]bool, result *ValidationResult) {
	for _, chronicle := range spec.Chronicles {
		owner := strings.TrimSpace(chronicle.OwnerType)
		switch owner {
		case "人物", model.ChronicleOwnerCharacter:
			if !characters[strings.ToLower(chronicle.OwnerName)] {
				result.Errors = append(result.Errors, Issue{Line: chronicle.Line, Field: "大事记/" + chronicle.Title, Code: CodeReferenceMissing, Message: fmt.Sprintf("人物「%s」不存在", chronicle.OwnerName)})
			}
		case "势力", model.ChronicleOwnerFaction:
			if !factions[strings.ToLower(chronicle.OwnerName)] {
				result.Errors = append(result.Errors, Issue{Line: chronicle.Line, Field: "大事记/" + chronicle.Title, Code: CodeReferenceMissing, Message: fmt.Sprintf("势力「%s」不存在", chronicle.OwnerName)})
			}
		default:
			result.Errors = append(result.Errors, Issue{Line: chronicle.Line, Field: "大事记/" + chronicle.Title + "/归属", Code: CodeInvalid, Message: "归属仅支持人物或势力"})
		}
		text := strings.TrimSpace(chronicle.Text)
		if text == "" {
			result.Errors = append(result.Errors, Issue{Line: chronicle.Line, Field: "大事记/" + chronicle.Title + "/内容", Code: CodeFieldRequired, Message: "大事记内容不能为空"})
		} else if len([]rune(text)) > model.MaxChronicleRunes {
			result.Errors = append(result.Errors, Issue{Line: chronicle.Line, Field: "大事记/" + chronicle.Title + "/内容", Code: CodeInvalid, Message: fmt.Sprintf("大事记不能超过 %d 个字符", model.MaxChronicleRunes)})
		}
		if chronicle.TimeLabel != "" && len([]rune(chronicle.TimeLabel)) > model.MaxChronicleTimeLabel {
			result.Errors = append(result.Errors, Issue{Line: chronicle.Line, Field: "大事记/" + chronicle.Title + "/时间", Code: CodeInvalid, Message: "时间文本过长"})
		}
	}
}
