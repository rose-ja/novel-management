package imports

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	sectionWork       = "作品信息"
	sectionCategories = "体系类别"
	sectionNodes      = "体系节点"
	sectionFactions   = "势力"
	sectionLocations  = "地点"
	sectionCharacters = "人物"
	sectionRankings   = "榜单"
	sectionChronicles = "大事记"
)

var sectionOrder = []string{
	sectionWork, sectionCategories, sectionNodes, sectionFactions,
	sectionLocations, sectionCharacters, sectionRankings, sectionChronicles,
}

type record struct {
	line   int
	title  string
	fields map[string][]string
	last   string
}

// Parse 解析固定模板 v1.0；语法与结构错误写入 issues，供预览展示。
func Parse(data []byte) (*Spec, []Issue) {
	spec := &Spec{Characters: []CharacterSpec{}}
	var issues []Issue
	text := strings.TrimPrefix(string(data), "\uFEFF")
	lines := strings.Split(text, "\n")

	section := ""
	var current *record
	var inComment bool

	finalize := func() {
		if current == nil {
			return
		}
		appendRecord(spec, current, section, &issues)
		current = nil
	}

	for index, raw := range lines {
		lineNo := index + 1
		line := strings.TrimSpace(raw)

		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}
		if strings.HasPrefix(line, "<!--") {
			if !strings.Contains(line, "-->") {
				inComment = true
			}
			continue
		}
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "## ") {
			finalize()
			name := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if !knownSection(name) {
				issues = append(issues, Issue{Line: lineNo, Field: "section", Code: CodeUnknownSection, Message: fmt.Sprintf("无法识别区块「%s」", name)})
				section = "unknown"
				continue
			}
			section = name
			continue
		}
		if strings.HasPrefix(line, "# ") {
			continue
		}

		if section == "" {
			if key, value, ok := splitField(line); ok && key == "模板版本" {
				spec.TemplateVersion = value
				continue
			}
			issues = append(issues, Issue{Line: lineNo, Field: "模板版本", Code: CodeFieldRequired, Message: "模板版本必须位于首个区块之前或作品信息中"})
			continue
		}
		if section == "unknown" {
			continue
		}

		switch section {
		case sectionWork:
			parseWorkLine(spec, line, lineNo, &issues)
		case sectionCategories:
			parseCategoryLine(spec, line, lineNo, &issues)
		case sectionNodes:
			parseNodeLine(spec, line, lineNo, &issues)
		default:
			if strings.HasPrefix(line, "### ") {
				finalize()
				current = &record{
					line:   lineNo,
					title:  strings.TrimSpace(strings.TrimPrefix(line, "### ")),
					fields: map[string][]string{},
				}
				continue
			}
			if current == nil {
				issues = append(issues, Issue{Line: lineNo, Field: section, Code: CodeInvalid, Message: "实体字段必须写在「### 名称」之下"})
				continue
			}
			if strings.HasPrefix(line, "| ") {
				if current.last == "" {
					issues = append(issues, Issue{Line: lineNo, Field: section, Code: CodeInvalid, Message: "续行前缺少字段"})
					continue
				}
				values := current.fields[current.last]
				values[len(values)-1] += "\n" + strings.TrimSpace(strings.TrimPrefix(line, "| "))
				current.fields[current.last] = values
				continue
			}
			key, value, ok := splitField(line)
			if !ok {
				issues = append(issues, Issue{Line: lineNo, Field: section, Code: CodeInvalid, Message: "字段格式应为「字段：值」"})
				continue
			}
			current.fields[key] = append(current.fields[key], value)
			current.last = key
			if key == "模板版本" {
				spec.TemplateVersion = value
			}
		}
	}
	finalize()

	if strings.TrimSpace(spec.TemplateVersion) == "" {
		issues = append(issues, Issue{Line: 1, Field: "模板版本", Code: CodeTemplateVersion, Message: "缺少模板版本，当前仅支持 v1.0"})
	} else if strings.TrimSpace(spec.TemplateVersion) != TemplateVersion {
		issues = append(issues, Issue{Line: 1, Field: "模板版本", Code: CodeTemplateVersion, Message: fmt.Sprintf("模板版本 %s 不受支持，当前仅支持 v1.0", spec.TemplateVersion)})
	}
	return spec, issues
}

func knownSection(name string) bool {
	for _, item := range sectionOrder {
		if item == name {
			return true
		}
	}
	return false
}

func splitField(line string) (string, string, bool) {
	index := strings.IndexAny(line, "：:")
	if index <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:index])
	rest := line[index:]
	if strings.HasPrefix(rest, "：") {
		rest = rest[len("："):]
	} else {
		rest = rest[1:]
	}
	return key, strings.TrimSpace(rest), true
}

func emptyValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "-" || value == "—" {
		return ""
	}
	return value
}

func splitList(value string) []string {
	value = strings.ReplaceAll(value, ";", "；")
	parts := strings.Split(value, "；")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitRelation(value string) RelationSpec {
	value = emptyValue(value)
	if index := strings.Index(value, "|"); index >= 0 {
		return RelationSpec{
			Name: strings.TrimSpace(value[:index]),
			Note: strings.TrimSpace(value[index+1:]),
		}
	}
	return RelationSpec{Name: value}
}

func parseWorkLine(spec *Spec, line string, lineNo int, issues *[]Issue) {
	key, value, ok := splitField(line)
	if !ok {
		*issues = append(*issues, Issue{Line: lineNo, Field: sectionWork, Code: CodeInvalid, Message: "作品信息应为「字段：值」"})
		return
	}
	switch key {
	case "模板版本":
		spec.TemplateVersion = value
	case "名称":
		spec.Work.Name = emptyValue(value)
		spec.Work.Line = lineNo
	case "简介":
		spec.Work.Description = emptyValue(value)
	default:
		*issues = append(*issues, Issue{Line: lineNo, Field: sectionWork + "/" + key, Code: CodeUnknownField, Message: fmt.Sprintf("未知作品字段「%s」", key)})
	}
}

func parseCategoryLine(spec *Spec, line string, lineNo int, issues *[]Issue) {
	value := line
	if key, rest, ok := splitField(line); ok && key == "体系类别" {
		value = rest
	}
	for _, item := range splitList(value) {
		builtin := false
		name := strings.TrimSpace(item)
		for _, marker := range []string{"（内置）", "(内置)"} {
			if strings.HasSuffix(name, marker) {
				name = strings.TrimSpace(strings.TrimSuffix(name, marker))
				builtin = true
			}
		}
		if name == "" {
			continue
		}
		spec.Categories = append(spec.Categories, CategorySpec{Name: name, Builtin: builtin, Line: lineNo})
	}
}

func parseNodeLine(spec *Spec, line string, lineNo int, issues *[]Issue) {
	if !strings.HasPrefix(line, "[") {
		*issues = append(*issues, Issue{Line: lineNo, Field: sectionNodes, Code: CodeInvalid, Message: "体系节点格式应为「[类别] 父 > 子」"})
		return
	}
	end := strings.Index(line, "]")
	if end < 0 {
		*issues = append(*issues, Issue{Line: lineNo, Field: sectionNodes, Code: CodeInvalid, Message: "体系节点缺少类别右括号"})
		return
	}
	category := strings.TrimSpace(line[1:end])
	rest := strings.TrimSpace(line[end+1:])
	parts := strings.Split(rest, ">")
	if len(parts) > 2 {
		*issues = append(*issues, Issue{Line: lineNo, Field: sectionNodes, Code: CodeNodeLevel, Message: "体系节点最多两级"})
		return
	}
	node := NodeSpec{Category: category, Line: lineNo}
	if len(parts) == 1 {
		node.Name = strings.TrimSpace(parts[0])
	} else {
		node.Parent = strings.TrimSpace(parts[0])
		node.Name = strings.TrimSpace(parts[1])
	}
	if category == "" || node.Name == "" {
		*issues = append(*issues, Issue{Line: lineNo, Field: sectionNodes, Code: CodeInvalid, Message: "体系节点缺少类别或名称"})
		return
	}
	spec.Nodes = append(spec.Nodes, node)
}

func appendRecord(spec *Spec, rec *record, section string, issues *[]Issue) {
	switch section {
	case sectionFactions:
		appendFaction(spec, rec, issues)
	case sectionLocations:
		appendLocation(spec, rec, issues)
	case sectionCharacters:
		appendCharacter(spec, rec, issues)
	case sectionRankings:
		appendRanking(spec, rec, issues)
	case sectionChronicles:
		appendChronicle(spec, rec, issues)
	}
}

func firstValue(rec *record, key string) string {
	values := rec.fields[key]
	if len(values) == 0 {
		return ""
	}
	return emptyValue(values[0])
}

func rejectUnknown(rec *record, allowed map[string]bool, field string, issues *[]Issue) {
	for key := range rec.fields {
		if !allowed[key] {
			*issues = append(*issues, Issue{Line: rec.line, Field: field + "/" + key, Code: CodeUnknownField, Message: fmt.Sprintf("未知字段「%s」", key)})
		}
	}
}

func appendFaction(spec *Spec, rec *record, issues *[]Issue) {
	rejectUnknown(rec, map[string]bool{"上级": true, "首领": true, "简介": true}, sectionFactions, issues)
	name := strings.TrimSpace(rec.title)
	if name == "" {
		*issues = append(*issues, Issue{Line: rec.line, Field: sectionFactions + "/名称", Code: CodeFieldRequired, Message: "势力名称不能为空"})
		return
	}
	spec.Factions = append(spec.Factions, FactionSpec{
		Name:        name,
		Parent:      firstValue(rec, "上级"),
		Leader:      firstValue(rec, "首领"),
		Description: firstValue(rec, "简介"),
		Line:        rec.line,
	})
}

func appendLocation(spec *Spec, rec *record, issues *[]Issue) {
	rejectUnknown(rec, map[string]bool{"上级": true, "类型": true, "简介": true, "势力": true}, sectionLocations, issues)
	name := strings.TrimSpace(rec.title)
	if name == "" {
		*issues = append(*issues, Issue{Line: rec.line, Field: sectionLocations + "/名称", Code: CodeFieldRequired, Message: "地点名称不能为空"})
		return
	}
	location := LocationSpec{
		Name:        name,
		Parent:      firstValue(rec, "上级"),
		TypeText:    firstValue(rec, "类型"),
		Description: firstValue(rec, "简介"),
		Line:        rec.line,
	}
	for _, raw := range rec.fields["势力"] {
		for _, item := range splitList(raw) {
			relation := splitRelation(item)
			relation.Line = rec.line
			location.Factions = append(location.Factions, relation)
		}
	}
	spec.Locations = append(spec.Locations, location)
}

func appendCharacter(spec *Spec, rec *record, issues *[]Issue) {
	name := strings.TrimSpace(rec.title)
	if name == "" {
		*issues = append(*issues, Issue{Line: rec.line, Field: sectionCharacters + "/名称", Code: CodeFieldRequired, Message: "人物名称不能为空"})
		return
	}
	character := CharacterSpec{
		Name:        name,
		Description: firstValue(rec, "简介"),
		Faction:     firstValue(rec, "势力"),
		SystemRefs:  map[string]string{},
		Extras:      map[string]string{},
		Image:       firstValue(rec, "图片"),
		Line:        rec.line,
	}
	for _, raw := range rec.fields["别名"] {
		character.Aliases = append(character.Aliases, splitList(raw)...)
	}
	for _, raw := range rec.fields["称号"] {
		character.Titles = append(character.Titles, splitList(raw)...)
	}
	for _, raw := range rec.fields["地点"] {
		for _, item := range splitList(raw) {
			relation := splitRelation(item)
			relation.Line = rec.line
			character.Locations = append(character.Locations, relation)
		}
	}
	for key, values := range rec.fields {
		switch key {
		case "别名", "称号", "简介", "势力", "地点", "图片":
			continue
		}
		character.Extras[key] = emptyValue(values[0])
	}
	spec.Characters = append(spec.Characters, character)
}

func appendRanking(spec *Spec, rec *record, issues *[]Issue) {
	name := strings.TrimSpace(rec.title)
	if name == "" {
		*issues = append(*issues, Issue{Line: rec.line, Field: sectionRankings + "/名称", Code: CodeFieldRequired, Message: "榜单名称不能为空"})
		return
	}
	ranking := RankingSpec{Name: name, Line: rec.line}
	for key, values := range rec.fields {
		switch {
		case key == "席位数":
			count, err := strconv.Atoi(emptyValue(values[0]))
			if err != nil {
				*issues = append(*issues, Issue{Line: rec.line, Field: sectionRankings + "/席位数", Code: CodeInvalid, Message: "席位数必须是整数"})
				continue
			}
			ranking.SeatCount = count
		case key == "席位":
			seat, err := parseSeat(emptyValue(values[0]))
			if err != nil {
				*issues = append(*issues, Issue{Line: rec.line, Field: sectionRankings + "/席位", Code: CodeInvalid, Message: err.Error()})
				continue
			}
			seat.Line = rec.line
			ranking.Seats = append(ranking.Seats, seat)
		case strings.HasPrefix(key, "第") && strings.HasSuffix(key, "席"):
			number, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(key, "第"), "席"))
			if err != nil || number < 1 {
				*issues = append(*issues, Issue{Line: rec.line, Field: sectionRankings + "/" + key, Code: CodeInvalid, Message: "席位号必须是正整数"})
				continue
			}
			seat := splitSeatValue(number, emptyValue(values[0]))
			seat.Line = rec.line
			ranking.Seats = append(ranking.Seats, seat)
		default:
			*issues = append(*issues, Issue{Line: rec.line, Field: sectionRankings + "/" + key, Code: CodeUnknownField, Message: fmt.Sprintf("未知榜单字段「%s」", key)})
		}
	}
	spec.Rankings = append(spec.Rankings, ranking)
}

func parseSeat(value string) (SeatSpec, error) {
	parts := strings.Split(value, "|")
	if len(parts) < 2 {
		return SeatSpec{}, fmt.Errorf("席位格式应为「序号 | 人物 | 备注」")
	}
	number, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || number < 1 {
		return SeatSpec{}, fmt.Errorf("席位号必须是正整数")
	}
	return splitSeatValue(number, strings.Join(parts[1:], "|")), nil
}

func splitSeatValue(number int, value string) SeatSpec {
	parts := strings.SplitN(value, "|", 2)
	seat := SeatSpec{No: number, Character: strings.TrimSpace(parts[0])}
	if len(parts) == 2 {
		seat.Note = strings.TrimSpace(parts[1])
	}
	return seat
}

func appendChronicle(spec *Spec, rec *record, issues *[]Issue) {
	rejectUnknown(rec, map[string]bool{"归属": true, "对象": true, "时间": true, "内容": true}, sectionChronicles, issues)
	chronicle := ChronicleSpec{
		Title:     strings.TrimSpace(rec.title),
		OwnerType: firstValue(rec, "归属"),
		OwnerName: firstValue(rec, "对象"),
		TimeLabel: firstValue(rec, "时间"),
		Text:      firstValue(rec, "内容"),
		Line:      rec.line,
	}
	if chronicle.Text == "" {
		*issues = append(*issues, Issue{Line: rec.line, Field: sectionChronicles + "/内容", Code: CodeFieldRequired, Message: "大事记内容不能为空"})
	}
	spec.Chronicles = append(spec.Chronicles, chronicle)
}
