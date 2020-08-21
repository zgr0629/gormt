package model

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/zgr0629/gormt/data/view/cnf"

	"github.com/xxjwxc/public/mybigcamel"

	"github.com/zgr0629/gormt/data/config"
	"github.com/zgr0629/gormt/data/view/genfunc"
	"github.com/zgr0629/gormt/data/view/genstruct"
)

type _Model struct {
	info DBInfo
	pkg  *genstruct.GenPackage
}

// Generate build code string.生成代码
func Generate(info DBInfo) (out []GenOutInfo, m _Model) {
	m = _Model{
		info: info,
	}

	// struct
	var stt GenOutInfo
	stt.FileCtx = m.generate()
	if fn := config.GetOutFileName(); fn != "" {
		stt.FileName = fn
	} else {
		stt.FileName = info.DbName + ".go"
	}

	out = append(out, stt)
	// ------end

	// gen function
	if config.GetIsOutFunc() {
		out = append(out, m.generateFunc()...)
	}
	// -------------- end
	return
}

// GetPackage gen sturct on table
func (m *_Model) GetPackage() genstruct.GenPackage {
	if m.pkg == nil {
		var pkg genstruct.GenPackage
		pkg.SetPackage(m.info.PackageName) //package name
		for _, tab := range m.info.TabList {
			var sct genstruct.GenStruct
			sct.SetTableName(tab.Name)
			sct.SetStructName(getCamelName(tab.Name)) // Big hump.大驼峰
			sct.SetNotes(tab.Notes)
			sct.AddElement(m.genTableElement(tab.Em)...) // build element.构造元素
			sct.SetCreatTableStr(tab.SQLBuildStr)
			pkg.AddStruct(sct)
		}
		m.pkg = &pkg
	}

	return *m.pkg
}

func (m *_Model) generate() string {
	m.pkg = nil
	m.GetPackage()
	return m.pkg.Generate()
}

// genTableElement Get table columns and comments.获取表列及注释
func (m *_Model) genTableElement(cols []ColumusInfo) (el []genstruct.GenElement) {
	_tagGorm := config.GetDBTag()
	_tagJSON := config.GetURLTag()

	for _, v := range cols {
		var tmp genstruct.GenElement
		if strings.EqualFold(v.Type, "gorm.Model") { // gorm model
			tmp.SetType(v.Type) //
		} else {
			tmp.SetName(getCamelName(v.Name))
			tmp.SetNotes(v.Notes)
			tmp.SetType(getTypeName(v.Type))
			for _, v1 := range v.Index {
				switch v1.Key {
				// case ColumusKeyDefault:
				case ColumusKeyPrimary: // primary key.主键
					tmp.AddTag(_tagGorm, "primary_key")
				case ColumusKeyUnique: // unique key.唯一索引
					tmp.AddTag(_tagGorm, "unique")
				case ColumusKeyIndex: // index key.复合索引
					tmp.AddTag(_tagGorm, getUninStr("index", ":", v1.KeyName))
				case ColumusKeyUniqueIndex: // unique index key.唯一复合索引
					tmp.AddTag(_tagGorm, getUninStr("unique_index", ":", v1.KeyName))
				}
			}
		}
		if len(v.Name) > 0 {
			// not simple output
			if !config.GetSimple() {
				tmp.AddTag(_tagGorm, "column:"+v.Name)
				tmp.AddTag(_tagGorm, "type:"+v.Type)
				if !v.IsNull {
					tmp.AddTag(_tagGorm, "not null")
				}
				if len(v.Default) > 0 {
					tmp.AddTag(_tagGorm, "default:"+v.Default)
				}
			}

			// json tag
			if config.GetIsWEBTag() {
				if strings.EqualFold(v.Name, "id") {
					tmp.AddTag(_tagJSON, "-")
				} else {
					tmp.AddTag(_tagJSON, mybigcamel.UnMarshal(v.Name))
				}
			}
		}

		el = append(el, tmp)

		// ForeignKey
		if config.GetIsForeignKey() && len(v.ForeignKeyList) > 0 {
			fklist := m.genForeignKey(v)
			el = append(el, fklist...)
		}
		// -----------end
	}

	return
}

// genForeignKey Get information about foreign key of table column.获取表列外键相关信息
func (m *_Model) genForeignKey(col ColumusInfo) (fklist []genstruct.GenElement) {
	_tagGorm := config.GetDBTag()
	_tagJSON := config.GetURLTag()

	for _, v := range col.ForeignKeyList {
		isMulti, isFind, notes := m.getColumusKeyMulti(v.TableName, v.ColumnName)
		if isFind {
			var tmp genstruct.GenElement
			tmp.SetNotes(notes)
			if isMulti {
				tmp.SetName(getCamelName(v.TableName) + "List")
				tmp.SetType("[]" + getCamelName(v.TableName))
			} else {
				tmp.SetName(getCamelName(v.TableName))
				tmp.SetType(getCamelName(v.TableName))
			}

			tmp.AddTag(_tagGorm, "association_foreignkey:"+col.Name)
			tmp.AddTag(_tagGorm, "foreignkey:"+v.ColumnName)

			// json tag
			if config.GetIsWEBTag() {
				tmp.AddTag(_tagJSON, mybigcamel.UnMarshal(v.TableName)+"_list")
			}

			fklist = append(fklist, tmp)
		}
	}

	return
}

func (m *_Model) getColumusKeyMulti(tableName, col string) (isMulti bool, isFind bool, notes string) {
	var haveGomod bool
	for _, v := range m.info.TabList {
		if strings.EqualFold(v.Name, tableName) {
			for _, v1 := range v.Em {
				if strings.EqualFold(v1.Name, col) {
					for _, v2 := range v1.Index {
						switch v2.Key {
						case ColumusKeyPrimary, ColumusKeyUnique, ColumusKeyUniqueIndex: // primary key unique key . 主键，唯一索引
							{
								return false, true, v.Notes
							}
							// case ColumusKeyIndex: // index key. 复合索引
							// 	{
							// 		isMulti = true
							// 	}
						}
					}
					return true, true, v.Notes
				} else if strings.EqualFold(v1.Type, "gorm.Model") {
					haveGomod = true
					notes = v.Notes
				}
			}
			break
		}
	}

	// default gorm.Model
	if haveGomod {
		if strings.EqualFold(col, "id") {
			return false, true, notes
		}

		if strings.EqualFold(col, "created_at") ||
			strings.EqualFold(col, "updated_at") ||
			strings.EqualFold(col, "deleted_at") {
			return true, true, notes
		}
	}

	return false, false, ""
	// -----------------end
}

// ///////////////////////// func
func (m *_Model) generateFunc() (genOut []GenOutInfo) {
	// getn base
	tmpl, err := template.New("gen_base").Parse(genfunc.GetGenBaseTemp())
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, m.info)
	genOut = append(genOut, GenOutInfo{
		FileName: "gen.base.go",
		FileCtx:  buf.String(),
	})
	//tools.WriteFile(outDir+"gen_router.go", []string{buf.String()}, true)
	// -------end------

	for _, tab := range m.info.TabList {
		var pkg genstruct.GenPackage
		pkg.SetPackage(m.info.PackageName) //package name
		pkg.AddImport(`"github.com/jinzhu/gorm"`)
		pkg.AddImport(`"fmt"`)

		data := funDef{
			StructName: getCamelName(tab.Name),
			TableName:  tab.Name,
		}

		var primary, unique, uniqueIndex, index []FList
		for _, el := range tab.Em {
			if strings.EqualFold(el.Type, "gorm.Model") {
				data.Em = append(data.Em, getGormModelElement()...)
				pkg.AddImport(`"time"`)
				buildFList(&primary, ColumusKeyPrimary, "", "int64", "id")
			} else {
				typeName := getTypeName(el.Type)
				isMulti := true
				for _, v1 := range el.Index {
					switch v1.Key {
					// case ColumusKeyDefault:
					case ColumusKeyPrimary: // primary key.主键
						isMulti = false
						buildFList(&primary, ColumusKeyPrimary, "", typeName, el.Name)
					case ColumusKeyUnique: // unique key.唯一索引
						isMulti = false
						buildFList(&unique, ColumusKeyUnique, "", typeName, el.Name)
					case ColumusKeyIndex: // index key.复合索引
						buildFList(&index, ColumusKeyIndex, v1.KeyName, typeName, el.Name)
					case ColumusKeyUniqueIndex: // unique index key.唯一复合索引
						isMulti = false
						buildFList(&uniqueIndex, ColumusKeyUniqueIndex, v1.KeyName, typeName, el.Name)
					}
				}

				data.Em = append(data.Em, EmInfo{
					IsMulti:       isMulti,
					Notes:         el.Notes,
					Type:          typeName, // Type.类型标记
					ColName:       el.Name,
					ColStructName: getCamelName(el.Name),
				})
				if v2, ok := cnf.EImportsHead[typeName]; ok {
					if len(v2) > 0 {
						pkg.AddImport(v2)
					}
				}
			}

			// 外键列表
			for _, v := range el.ForeignKeyList {
				isMulti, isFind, notes := m.getColumusKeyMulti(v.TableName, v.ColumnName)
				if isFind {
					var info PreloadInfo
					info.IsMulti = isMulti
					info.Notes = notes
					info.ForeignkeyTableName = v.TableName
					info.ForeignkeyCol = v.ColumnName
					info.ForeignkeyStructName = getCamelName(v.TableName)
					info.ColName = el.Name
					info.ColStructName = getCamelName(el.Name)
					data.PreloadList = append(data.PreloadList, info)
				}
			}
			// ---------end--
		}

		data.Primay = append(data.Primay, primary...)
		data.Primay = append(data.Primay, unique...)
		data.Primay = append(data.Primay, uniqueIndex...)
		data.Index = append(data.Index, index...)

		tmpl, err := template.New("gen_logic").
			Funcs(template.FuncMap{"GenPreloadList": GenPreloadList, "GenFListIndex": GenFListIndex}).
			Parse(genfunc.GetGenLogicTemp())
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		tmpl.Execute(&buf, data)

		pkg.AddFuncStr(buf.String())
		genOut = append(genOut, GenOutInfo{
			FileName: fmt.Sprintf(m.info.DbName+".gen.%v.go", tab.Name),
			FileCtx:  pkg.Generate(),
		})
	}

	return
}
