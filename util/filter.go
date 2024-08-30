package util

import (
	"regexp"
	"strings"
)

func matchPatterns(filename string, filters []FilterOptionType) bool {
	if len(filters) == 0 {
		return true
	}

	files := []string{filename}
	vsf := matchFiltersForStrs(files, filters)

	if len(vsf) > 0 {
		return true
	}
	return false
}

func matchFiltersForStrs(strs []string, filters []FilterOptionType) []string {
	if len(filters) == 0 {
		return strs
	}

	vsf := make([]string, 0)

	for _, str := range strs {
		if matchFiltersForStr(str, filters) {
			vsf = append(vsf, str)
		}
	}

	return vsf
}

func matchFiltersForStr(str string, filters []FilterOptionType) bool {
	if len(filters) == 0 {
		return true
	}

	var res bool
	if filters[0].name == IncludePrompt {
		res = filterSingleStr(str, filters[0].pattern, true)
	} else {
		res = filterSingleStr(str, filters[0].pattern, false)
	}
	for _, filter := range filters[1:] {
		if filter.name == IncludePrompt {
			res = res || filterSingleStr(str, filter.pattern, true)
		} else {
			res = res && filterSingleStr(str, filter.pattern, false)
		}
	}

	return res
}

func filterSingleStr(v, p string, include bool) bool {
	//_, name := filepath.Split(v)
	res, _ := regexp.MatchString(p, v)

	if include {
		return res
	} else {
		return !res
	}
}

func GetFilter(include, exclude string) (bool, []FilterOptionType) {
	filters := make([]FilterOptionType, 0)

	if include != "" {
		ok, filter := createFilter(IncludePrompt, include)
		if !ok {
			return false, filters
		}
		filters = append(filters, filter)
	}

	if exclude != "" {
		ok, filter := createFilter(ExcludePrompt, exclude)
		if !ok {
			return false, filters
		}
		filters = append(filters, filter)
	}

	return true, filters
}

func createFilter(name, pattern string) (bool, FilterOptionType) {
	var filter FilterOptionType
	filter.name = name
	filter.pattern = strings.Replace(pattern, "[!", "[^", -1)
	//dir, _ := filepath.Split(filter.pattern)
	//if dir != "" {
	//	return false, filter
	//}
	return true, filter
}

func cosObjectMatchPatterns(object string, filters []FilterOptionType) bool {
	if len(filters) == 0 {
		return true
	}

	return matchPatterns(object, filters)
}
