package model

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/xxjwxc/public/mybigcamel"
	"github.com/zgr0629/gormt/data/config"
	"github.com/zgr0629/gormt/data/view/cnf"
	"github.com/zgr0629/gormt/data/view/genfunc"
)

// getCamelName Big Hump or Capital Letter.大驼峰或者首字母大写
func getCamelName(name string) string {
	if !config.GetSingularTable() { // If the table name plural is globally disabled.如果全局禁用表名复数
		return mybigcamel.Marshal(strings.TrimSuffix(name, "s"))
	}

	return mybigcamel.Marshal(name)
}

// titleCase title case.首字母大写
func titleCase(name string) string {
	vv := []rune(name)
	if len(vv) > 0 {
		if bool(vv[0] >= 'a' && vv[0] <= 'z') { // title case.首字母大写
			vv[0] -= 32
		}
	}

	return string(vv)
}

// getTypeName Type acquisition filtering.类型获取过滤
func getTypeName(name string) string {
	// Precise matching first.先精确匹配
	if v, ok := cnf.TypeMysqlDicMp[name]; ok {
		return v
	}

	// Fuzzy Regular Matching.模糊正则匹配
	for k, v := range cnf.TypeMysqlMatchMp {
		if ok, _ := regexp.MatchString(k, name); ok {
			return v
		}
	}

	panic(fmt.Sprintf("type (%v) not match in any way.maybe need to add on (https://github.com/xxjwxc/gormt/blob/master/data/view/cnf/def.go)", name))
}

func getUninStr(left, middle, right string) string {
	re := left
	if len(right) > 0 {
		re = left + middle + right
	}
	return re
}

func getGormModelElement() []EmInfo {
	var result []EmInfo
	result = append(result, EmInfo{
		IsMulti:       false,
		Notes:         "Primary key",
		Type:          "int64", // Type.类型标记
		ColName:       "id",
		ColStructName: "ID",
	})
	result = append(result, EmInfo{
		IsMulti:       false,
		Notes:         "created time",
		Type:          "time.Time", // Type.类型标记
		ColName:       "created_at",
		ColStructName: "CreatedAt",
	})

	result = append(result, EmInfo{
		IsMulti:       false,
		Notes:         "updated at",
		Type:          "time.Time", // Type.类型标记
		ColName:       "updated_at",
		ColStructName: "UpdatedAt",
	})

	result = append(result, EmInfo{
		IsMulti:       false,
		Notes:         "deleted time",
		Type:          "time.Time", // Type.类型标记
		ColName:       "deleted_at",
		ColStructName: "DeletedAt",
	})
	return result
}

func buildFList(list *[]FList, key ColumusKey, keyName, tp, colName string) {
	for i := 0; i < len(*list); i++ {
		if (*list)[i].KeyName == keyName {
			(*list)[i].Kem = append((*list)[i].Kem, FEm{
				Type:          tp,
				ColName:       colName,
				ColStructName: getCamelName(colName),
			})
			return
		}
	}
	// 没有 添加一个
	*list = append(*list, FList{
		Key:     key,
		KeyName: keyName,
		Kem: []FEm{FEm{
			Type:          tp,
			ColName:       colName,
			ColStructName: getCamelName(colName),
		}},
	})
}

// GenPreloadList 生成list
func GenPreloadList(list []PreloadInfo, multi bool) string {
	if len(list) > 0 {
		tmpl, err := template.New("gen_preload").Parse(genfunc.GetGenPreloadTemp(multi))
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		tmpl.Execute(&buf, list)

		return buf.String()
	}

	return ""
}

// GenFListIndex 生成list status(1:获取函数名,2:获取参数列表,3:获取sql case,4:值列表)
func GenFListIndex(info FList, status int) string {
	switch status {
	case 1: // 1:获取函数名
		{
			return widthFunctionName(info)
		}
	case 2: // 2:获取参数列表
		{
			var strs []string
			for _, v := range info.Kem {
				strs = append(strs, fmt.Sprintf("%v %v ", v.ColStructName, v.Type))
			}
			return strings.Join(strs, ",")
		}
	case 3: // 3:获取sql case,
		{
			var strs []string
			for _, v := range info.Kem {
				strs = append(strs, fmt.Sprintf("%v = ?", v.ColName))
			}
			return strings.Join(strs, " AND ")
		}
	case 4: // 4:值列表
		{
			var strs []string
			for _, v := range info.Kem {
				strs = append(strs, v.ColStructName)
			}
			return strings.Join(strs, " , ")
		}
	}

	return ""
}

func widthFunctionName(info FList) string {
	switch info.Key {
	// case ColumusKeyDefault:
	case ColumusKeyPrimary: // primary key.主键
		return "FetchByPrimaryKey"
	case ColumusKeyUnique: // unique key.唯一索引
		return "FetchByUnique"
	case ColumusKeyIndex: // index key.复合索引
		return "FetchBy" + getCamelName(info.KeyName) + "Index"
	case ColumusKeyUniqueIndex: // unique index key.唯一复合索引
		return "FetchBy" + getCamelName(info.KeyName) + "UniqueIndex"
	}

	return ""
}
