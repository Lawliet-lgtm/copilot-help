package processor

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// ============================================================
// PDF 对象类型
// ============================================================

// PdfObjectType PDF对象类型
type PdfObjectType int

const (
	PdfNull PdfObjectType = iota
	PdfBool
	PdfInt
	PdfReal
	PdfString
	PdfName
	PdfArray
	PdfDict
	PdfStream
	PdfRef
)

// PdfObject PDF对象接口
type PdfObject interface {
	Type() PdfObjectType
	String() string
}

// PdfNullObject 空对象
type PdfNullObject struct{}

func (o PdfNullObject) Type() PdfObjectType { return PdfNull }
func (o PdfNullObject) String() string      { return "null" }

// PdfBoolObject 布尔对象
type PdfBoolObject struct {
	Value bool
}

func (o PdfBoolObject) Type() PdfObjectType { return PdfBool }
func (o PdfBoolObject) String() string      { return fmt.Sprintf("%v", o.Value) }

// PdfIntObject 整数对象
type PdfIntObject struct {
	Value int64
}

func (o PdfIntObject) Type() PdfObjectType { return PdfInt }
func (o PdfIntObject) String() string      { return fmt.Sprintf("%d", o.Value) }

// PdfRealObject 实数对象
type PdfRealObject struct {
	Value float64
}

func (o PdfRealObject) Type() PdfObjectType { return PdfReal }
func (o PdfRealObject) String() string      { return fmt.Sprintf("%f", o.Value) }

// PdfStringObject 字符串对象
type PdfStringObject struct {
	Value string
	IsHex bool
}

func (o PdfStringObject) Type() PdfObjectType { return PdfString }
func (o PdfStringObject) String() string      { return o.Value }

// PdfNameObject 名称对象
type PdfNameObject struct {
	Value string
}

func (o PdfNameObject) Type() PdfObjectType { return PdfName }
func (o PdfNameObject) String() string      { return "/" + o.Value }

// PdfArrayObject 数组对象
type PdfArrayObject struct {
	Items []PdfObject
}

func (o PdfArrayObject) Type() PdfObjectType { return PdfArray }
func (o PdfArrayObject) String() string {
	var items []string
	for _, item := range o.Items {
		items = append(items, item.String())
	}
	return "[" + strings.Join(items, " ") + "]"
}

// PdfDictObject 字典对象
type PdfDictObject struct {
	Dict map[string]PdfObject
}

func (o PdfDictObject) Type() PdfObjectType { return PdfDict }
func (o PdfDictObject) String() string {
	var items []string
	for k, v := range o.Dict {
		items = append(items, "/"+k+" "+v.String())
	}
	return "<< " + strings.Join(items, " ") + " >>"
}

// Get 获取字典中的值
func (o PdfDictObject) Get(key string) PdfObject {
	if o.Dict == nil {
		return nil
	}
	return o.Dict[key]
}

// GetString 获取字符串值
func (o PdfDictObject) GetString(key string) string {
	obj := o.Get(key)
	if obj == nil {
		return ""
	}
	switch v := obj.(type) {
	case PdfStringObject:
		return v.Value
	case PdfNameObject:
		return v.Value
	default:
		return obj.String()
	}
}

// GetInt 获取整数值
func (o PdfDictObject) GetInt(key string) int64 {
	obj := o.Get(key)
	if obj == nil {
		return 0
	}
	if v, ok := obj.(PdfIntObject); ok {
		return v.Value
	}
	return 0
}

// GetReal 获取实数值
func (o PdfDictObject) GetReal(key string) float64 {
	obj := o.Get(key)
	if obj == nil {
		return 0
	}
	switch v := obj.(type) {
	case PdfRealObject:
		return v.Value
	case PdfIntObject:
		return float64(v.Value)
	}
	return 0
}

// GetArray 获取数组值
func (o PdfDictObject) GetArray(key string) []PdfObject {
	obj := o.Get(key)
	if obj == nil {
		return nil
	}
	if v, ok := obj.(PdfArrayObject); ok {
		return v.Items
	}
	return nil
}

// GetDict 获取字典值
func (o PdfDictObject) GetDict(key string) *PdfDictObject {
	obj := o.Get(key)
	if obj == nil {
		return nil
	}
	if v, ok := obj.(PdfDictObject); ok {
		return &v
	}
	return nil
}

// PdfStreamObject 流对象
type PdfStreamObject struct {
	Dict    PdfDictObject
	RawData []byte
}

func (o PdfStreamObject) Type() PdfObjectType { return PdfStream }
func (o PdfStreamObject) String() string      { return o.Dict.String() + " stream..." }

// GetDecodedData 获取解码后的流数据
func (o PdfStreamObject) GetDecodedData() ([]byte, error) {
	filter := o.Dict.GetString("Filter")

	switch filter {
	case "FlateDecode":
		return decodeFlate(o.RawData)
	case "":
		return o.RawData, nil
	default:
		// 不支持的编码，返回原始数据
		return o.RawData, nil
	}
}

// PdfRefObject 引用对象
type PdfRefObject struct {
	ObjNum int
	GenNum int
}

func (o PdfRefObject) Type() PdfObjectType { return PdfRef }
func (o PdfRefObject) String() string      { return fmt.Sprintf("%d %d R", o.ObjNum, o.GenNum) }

// ============================================================
// PDF 解析器
// ============================================================

// PdfParser PDF解析器
type PdfParser struct {
	data       []byte
	pos        int
	objects    map[int]PdfObject       // 对象号 -> 对象
	xrefTable  map[int]int64           // 对象号 -> 文件偏移
	trailer    *PdfDictObject          // trailer字典
	pageObjs   []PdfObject             // 页面对象列表
	fontObjs   map[string]*PdfDictObject // 字体对象
}

// NewPdfParser 创建PDF解析器
func NewPdfParser(data []byte) *PdfParser {
	return &PdfParser{
		data:      data,
		objects:   make(map[int]PdfObject),
		xrefTable: make(map[int]int64),
		fontObjs:  make(map[string]*PdfDictObject),
	}
}

// Parse 解析PDF文件
func (p *PdfParser) Parse() error {
	// 1. 验证PDF头
	if err := p.parseHeader(); err != nil {
		return err
	}

	// 2. 解析xref和trailer
	if err := p.parseXrefAndTrailer(); err != nil {
		return err
	}

	// 3. 加载页面对象
	if err := p.loadPages(); err != nil {
		return err
	}

	return nil
}

// parseHeader 解析PDF头
func (p *PdfParser) parseHeader() error {
	if len(p.data) < 8 {
		return fmt.Errorf("文件太小，不是有效的PDF")
	}

	header := string(p.data[:8])
	if !strings.HasPrefix(header, "%PDF-") {
		return fmt.Errorf("不是有效的PDF文件")
	}

	return nil
}

// parseXrefAndTrailer 解析交叉引用表和trailer
func (p *PdfParser) parseXrefAndTrailer() error {
	// 从文件末尾查找 startxref
	data := p.data
	
	// 查找 %%EOF
	eofPos := bytes.LastIndex(data, []byte("%%EOF"))
	if eofPos == -1 {
		return fmt.Errorf("未找到PDF结束标记")
	}

	// 查找 startxref
	startxrefPos := bytes.LastIndex(data[:eofPos], []byte("startxref"))
	if startxrefPos == -1 {
		return fmt.Errorf("未找到startxref")
	}

	// 读取xref偏移
	p.pos = startxrefPos + 9
	p.skipWhitespace()
	xrefOffset := p.readInt()

	// 解析xref表
	p.pos = int(xrefOffset)
	if err := p.parseXref(); err != nil {
		// 可能是xref流，尝试其他方式
		return p.parseXrefStream(int(xrefOffset))
	}

	// 解析trailer
	return p.parseTrailer()
}

// parseXref 解析传统xref表
func (p *PdfParser) parseXref() error {
	p.skipWhitespace()
	
	// 检查是否是 "xref"
	if !p.matchKeyword("xref") {
		return fmt.Errorf("期望xref关键字")
	}

	p.skipWhitespace()

	// 读取各个xref小节
	for {
		p.skipWhitespace()
		
		// 检查是否到达trailer
		if p.matchKeyword("trailer") {
			p.pos -= 7 // 回退，让parseTrailer处理
			break
		}

		// 读取起始对象号和数量
		startObj := p.readInt()
		p.skipWhitespace()
		count := p.readInt()
		p.skipWhitespace()

		// 读取条目
		for i := int64(0); i < count; i++ {
			offset := p.readInt()
			p.skipWhitespace()
			_ = p.readInt() // gen号
			p.skipWhitespace()

			// 读取标志 (n 或 f)
			flag := p.readByte()
			p.skipWhitespace()

			if flag == 'n' {
				p.xrefTable[int(startObj+i)] = offset
			}
		}
	}

	return nil
}

// parseXrefStream 解析xref流（PDF 1.5+）
func (p *PdfParser) parseXrefStream(offset int) error {
	p.pos = offset
	
	// 读取对象
	obj, err := p.readObject()
	if err != nil {
		return err
	}

	stream, ok := obj.(PdfStreamObject)
	if !ok {
		return fmt.Errorf("xref流不是流对象")
	}

	// 从流字典获取trailer信息
	p.trailer = &stream.Dict

	// 解析流内容以获取xref条目（简化处理）
	// 实际实现需要根据 W 数组解析

	return nil
}

// parseTrailer 解析trailer
func (p *PdfParser) parseTrailer() error {
	p.skipWhitespace()

	if !p.matchKeyword("trailer") {
		return fmt.Errorf("期望trailer关键字")
	}

	p.skipWhitespace()

	obj, err := p.readObject()
	if err != nil {
		return err
	}

	dict, ok := obj.(PdfDictObject)
	if !ok {
		return fmt.Errorf("trailer不是字典")
	}

	p.trailer = &dict
	return nil
}

// loadPages 加载页面对象
func (p *PdfParser) loadPages() error {
	if p.trailer == nil {
		return fmt.Errorf("trailer未加载")
	}

	// 获取Root对象
	rootRef := p.trailer.Get("Root")
	if rootRef == nil {
		return fmt.Errorf("未找到Root引用")
	}

	root, err := p.resolveRef(rootRef)
	if err != nil {
		return err
	}

	rootDict, ok := root.(PdfDictObject)
	if !ok {
		return fmt.Errorf("Root不是字典")
	}

	// 获取Pages对象
	pagesRef := rootDict.Get("Pages")
	if pagesRef == nil {
		return fmt.Errorf("未找到Pages引用")
	}

	pages, err := p.resolveRef(pagesRef)
	if err != nil {
		return err
	}

	pagesDict, ok := pages.(PdfDictObject)
	if !ok {
		return fmt.Errorf("Pages不是字典")
	}

	// 递归获取所有页面
	return p.collectPages(&pagesDict)
}

// collectPages 递归收集页面对象
func (p *PdfParser) collectPages(pagesDict *PdfDictObject) error {
	typeVal := pagesDict.GetString("Type")

	if typeVal == "Page" {
		p.pageObjs = append(p.pageObjs, *pagesDict)
		return nil
	}

	if typeVal == "Pages" {
		kids := pagesDict.GetArray("Kids")
		for _, kidRef := range kids {
			kid, err := p.resolveRef(kidRef)
			if err != nil {
				continue
			}
			kidDict, ok := kid.(PdfDictObject)
			if ok {
				if err := p.collectPages(&kidDict); err != nil {
					continue
				}
			}
		}
	}

	return nil
}

// GetObject 获取指定对象号的对象
func (p *PdfParser) GetObject(objNum int) (PdfObject, error) {
	// 检查缓存
	if obj, ok := p.objects[objNum]; ok {
		return obj, nil
	}

	// 从xref表获取偏移
	offset, ok := p.xrefTable[objNum]
	if !ok {
		return nil, fmt.Errorf("对象 %d 不存在", objNum)
	}

	// 读取对象
	p.pos = int(offset)
	obj, err := p.readIndirectObject()
	if err != nil {
		return nil, err
	}

	// 缓存
	p.objects[objNum] = obj
	return obj, nil
}

// resolveRef 解析引用
func (p *PdfParser) resolveRef(obj PdfObject) (PdfObject, error) {
	ref, ok := obj.(PdfRefObject)
	if !ok {
		return obj, nil
	}

	return p.GetObject(ref.ObjNum)
}

// GetPages 获取所有页面
func (p *PdfParser) GetPages() []PdfDictObject {
	var pages []PdfDictObject
	for _, obj := range p.pageObjs {
		if dict, ok := obj.(PdfDictObject); ok {
			pages = append(pages, dict)
		}
	}
	return pages
}

// GetPageCount 获取页面数量
func (p *PdfParser) GetPageCount() int {
	return len(p.pageObjs)
}

// ============================================================
// 低级解析方法
// ============================================================

// readIndirectObject 读取间接对象
func (p *PdfParser) readIndirectObject() (PdfObject, error) {
	p.skipWhitespace()

	// 读取对象号
	_ = p.readInt()
	p.skipWhitespace()

	// 读取生成号
	_ = p.readInt()
	p.skipWhitespace()

	// 期望 "obj"
	if !p.matchKeyword("obj") {
		return nil, fmt.Errorf("期望obj关键字")
	}

	p.skipWhitespace()

	// 读取对象内容
	obj, err := p.readObject()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()

	// 检查是否是流对象
	if dict, ok := obj.(PdfDictObject); ok {
		if p.matchKeyword("stream") {
			// 跳过换行
			if p.pos < len(p.data) && p.data[p.pos] == '\r' {
				p.pos++
			}
			if p.pos < len(p.data) && p.data[p.pos] == '\n' {
				p.pos++
			}

			// 读取流数据
			length := dict.GetInt("Length")
			if length <= 0 {
				length = 0
			}

			endPos := p.pos + int(length)
			if endPos > len(p.data) {
				endPos = len(p.data)
			}

			streamData := p.data[p.pos:endPos]
			p.pos = endPos

			return PdfStreamObject{
				Dict:    dict,
				RawData: streamData,
			}, nil
		}
	}

	return obj, nil
}

// readObject 读取对象
func (p *PdfParser) readObject() (PdfObject, error) {
	p.skipWhitespace()

	if p.pos >= len(p.data) {
		return nil, io.EOF
	}

	ch := p.data[p.pos]

	switch {
	case ch == '<':
		if p.pos+1 < len(p.data) && p.data[p.pos+1] == '<' {
			return p.readDict()
		}
		return p.readHexString()

	case ch == '(':
		return p.readLiteralString()

	case ch == '/':
		return p.readName()

	case ch == '[':
		return p.readArray()

	case ch == 't' || ch == 'f':
		return p.readBool()

	case ch == 'n':
		return p.readNull()

	case ch == '-' || ch == '+' || ch == '.' || (ch >= '0' && ch <= '9'):
		return p.readNumber()

	default:
		return nil, fmt.Errorf("未知的对象类型: %c", ch)
	}
}

// readDict 读取字典
func (p *PdfParser) readDict() (PdfDictObject, error) {
	dict := PdfDictObject{Dict: make(map[string]PdfObject)}

	// 跳过 <<
	p.pos += 2
	p.skipWhitespace()

	for {
		p.skipWhitespace()

		if p.pos >= len(p.data) {
			break
		}

		// 检查是否结束
		if p.data[p.pos] == '>' && p.pos+1 < len(p.data) && p.data[p.pos+1] == '>' {
			p.pos += 2
			break
		}

		// 读取键（名称）
		nameObj, err := p.readName()
		if err != nil {
			return dict, err
		}

		p.skipWhitespace()

		// 读取值
		value, err := p.readObject()
		if err != nil {
			return dict, err
		}

		// 检查是否是引用
		if intObj, ok := value.(PdfIntObject); ok {
			p.skipWhitespace()
			savedPos := p.pos
			genNum := p.readInt()
			p.skipWhitespace()
			if p.matchKeyword("R") {
				value = PdfRefObject{ObjNum: int(intObj.Value), GenNum: int(genNum)}
			} else {
				p.pos = savedPos
			}
		}

		dict.Dict[nameObj.Value] = value
	}

	return dict, nil
}

// readArray 读取数组
func (p *PdfParser) readArray() (PdfArrayObject, error) {
	arr := PdfArrayObject{}

	// 跳过 [
	p.pos++
	p.skipWhitespace()

	for {
		p.skipWhitespace()

		if p.pos >= len(p.data) {
			break
		}

		if p.data[p.pos] == ']' {
			p.pos++
			break
		}

		obj, err := p.readObject()
		if err != nil {
			return arr, err
		}

		// 检查是否是引用
		if intObj, ok := obj.(PdfIntObject); ok {
			p.skipWhitespace()
			savedPos := p.pos
			genNum := p.readInt()
			p.skipWhitespace()
			if p.matchKeyword("R") {
				obj = PdfRefObject{ObjNum: int(intObj.Value), GenNum: int(genNum)}
			} else {
				p.pos = savedPos
			}
		}

		arr.Items = append(arr.Items, obj)
	}

	return arr, nil
}

// readName 读取名称
func (p *PdfParser) readName() (PdfNameObject, error) {
	// 跳过 /
	p.pos++

	var name strings.Builder
	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if isWhitespace(ch) || isDelimiter(ch) {
			break
		}
		if ch == '#' && p.pos+2 < len(p.data) {
			// 十六进制转义
			hex := string(p.data[p.pos+1 : p.pos+3])
			if val, err := strconv.ParseInt(hex, 16, 8); err == nil {
				name.WriteByte(byte(val))
				p.pos += 3
				continue
			}
		}
		name.WriteByte(ch)
		p.pos++
	}

	return PdfNameObject{Value: name.String()}, nil
}

// readLiteralString 读取文字字符串
func (p *PdfParser) readLiteralString() (PdfStringObject, error) {
	// 跳过 (
	p.pos++

	var str strings.Builder
	depth := 1

	for p.pos < len(p.data) && depth > 0 {
		ch := p.data[p.pos]

		if ch == '\\' && p.pos+1 < len(p.data) {
			p.pos++
			escaped := p.data[p.pos]
			switch escaped {
			case 'n':
				str.WriteByte('\n')
			case 'r':
				str.WriteByte('\r')
			case 't':
				str.WriteByte('\t')
			case 'b':
				str.WriteByte('\b')
			case 'f':
				str.WriteByte('\f')
			case '(', ')', '\\':
				str.WriteByte(escaped)
			default:
				// 八进制
				if escaped >= '0' && escaped <= '7' {
					octal := string(escaped)
					for i := 0; i < 2 && p.pos+1 < len(p.data); i++ {
						next := p.data[p.pos+1]
						if next >= '0' && next <= '7' {
							octal += string(next)
							p.pos++
						} else {
							break
						}
					}
					if val, err := strconv.ParseInt(octal, 8, 8); err == nil {
						str.WriteByte(byte(val))
					}
				} else {
					str.WriteByte(escaped)
				}
			}
		} else if ch == '(' {
			depth++
			str.WriteByte(ch)
		} else if ch == ')' {
			depth--
			if depth > 0 {
				str.WriteByte(ch)
			}
		} else {
			str.WriteByte(ch)
		}
		p.pos++
	}

	return PdfStringObject{Value: str.String(), IsHex: false}, nil
}

// readHexString 读取十六进制字符串
func (p *PdfParser) readHexString() (PdfStringObject, error) {
	// 跳过 <
	p.pos++

	var hexChars strings.Builder
	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if ch == '>' {
			p.pos++
			break
		}
		if !isWhitespace(ch) {
			hexChars.WriteByte(ch)
		}
		p.pos++
	}

	hex := hexChars.String()
	if len(hex)%2 == 1 {
		hex += "0"
	}

	var result strings.Builder
	for i := 0; i < len(hex); i += 2 {
		if val, err := strconv.ParseInt(hex[i:i+2], 16, 8); err == nil {
			result.WriteByte(byte(val))
		}
	}

	return PdfStringObject{Value: result.String(), IsHex: true}, nil
}

// readNumber 读取数字
func (p *PdfParser) readNumber() (PdfObject, error) {
	var numStr strings.Builder
	hasDecimal := false

	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if ch == '.' {
			if hasDecimal {
				break
			}
			hasDecimal = true
			numStr.WriteByte(ch)
		} else if ch == '-' || ch == '+' {
			if numStr.Len() > 0 {
				break
			}
			numStr.WriteByte(ch)
		} else if ch >= '0' && ch <= '9' {
			numStr.WriteByte(ch)
		} else {
			break
		}
		p.pos++
	}

	str := numStr.String()
	if hasDecimal {
		val, _ := strconv.ParseFloat(str, 64)
		return PdfRealObject{Value: val}, nil
	}

	val, _ := strconv.ParseInt(str, 10, 64)
	return PdfIntObject{Value: val}, nil
}

// readBool 读取布尔值
func (p *PdfParser) readBool() (PdfBoolObject, error) {
	if p.matchKeyword("true") {
		return PdfBoolObject{Value: true}, nil
	}
	if p.matchKeyword("false") {
		return PdfBoolObject{Value: false}, nil
	}
	return PdfBoolObject{}, fmt.Errorf("无效的布尔值")
}

// readNull 读取null
func (p *PdfParser) readNull() (PdfNullObject, error) {
	if p.matchKeyword("null") {
		return PdfNullObject{}, nil
	}
	return PdfNullObject{}, fmt.Errorf("无效的null")
}

// ============================================================
// 辅助方法
// ============================================================

func (p *PdfParser) skipWhitespace() {
	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if ch == '%' {
			// 跳过注释
			for p.pos < len(p.data) && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' {
				p.pos++
			}
		} else if isWhitespace(ch) {
			p.pos++
		} else {
			break
		}
	}
}

func (p *PdfParser) readInt() int64 {
	var numStr strings.Builder
	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if ch >= '0' && ch <= '9' {
			numStr.WriteByte(ch)
			p.pos++
		} else if ch == '-' || ch == '+' {
			if numStr.Len() == 0 {
				numStr.WriteByte(ch)
				p.pos++
			} else {
				break
			}
		} else {
			break
		}
	}
	val, _ := strconv.ParseInt(numStr.String(), 10, 64)
	return val
}

func (p *PdfParser) readByte() byte {
	if p.pos < len(p.data) {
		ch := p.data[p.pos]
		p.pos++
		return ch
	}
	return 0
}

func (p *PdfParser) matchKeyword(keyword string) bool {
	if p.pos+len(keyword) > len(p.data) {
		return false
	}
	if string(p.data[p.pos:p.pos+len(keyword)]) == keyword {
		// 确保后面是分隔符或空白
		if p.pos+len(keyword) < len(p.data) {
			next := p.data[p.pos+len(keyword)]
			if !isWhitespace(next) && !isDelimiter(next) {
				return false
			}
		}
		p.pos += len(keyword)
		return true
	}
	return false
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == 0
}

func isDelimiter(ch byte) bool {
	return ch == '(' || ch == ')' || ch == '<' || ch == '>' ||
		ch == '[' || ch == ']' || ch == '{' || ch == '}' ||
		ch == '/' || ch == '%'
}

// decodeFlate 解压FlateDecode数据
func decodeFlate(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return data, err
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		return data, err
	}

	return decoded, nil
}

// ============================================================
// 文本解码辅助
// ============================================================

// decodePdfString 解码PDF字符串（处理编码）
func decodePdfString(s string) string {
	// 检查BOM
	if len(s) >= 2 {
		if s[0] == 0xFE && s[1] == 0xFF {
			// UTF-16BE
			return decodeUTF16BE([]byte(s[2:]))
		}
		if s[0] == 0xFF && s[1] == 0xFE {
			// UTF-16LE
			return decodeUTF16LE([]byte(s[2:]))
		}
	}

	// 假设是 PDFDocEncoding 或 Latin-1
	return s
}

// cMapPattern 用于提取CMap中的映射
var cMapPattern = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)

// parseCMap 解析ToUnicode CMap
func parseCMap(data []byte) map[uint16]rune {
	result := make(map[uint16]rune)

	matches := cMapPattern.FindAllSubmatch(data, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			srcHex := string(match[1])
			dstHex := string(match[2])

			srcVal, err1 := strconv.ParseUint(srcHex, 16, 16)
			dstVal, err2 := strconv.ParseUint(dstHex, 16, 32)

			if err1 == nil && err2 == nil {
				result[uint16(srcVal)] = rune(dstVal)
			}
		}
	}

	return result
}