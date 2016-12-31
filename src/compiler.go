package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
)

const kuuhaktokigou = " !=<>()|&+-*/%,[]^:;{}#\"\t\r\n"

type expFunc func(out *BytecodeExp, in *string) (BytecodeValue, error)
type scFunc func(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, ihp bool) (StateController, error)
type Compiler struct {
	cmdl     *CommandList
	maeOp    string
	usiroOp  bool
	norange  bool
	token    string
	playerNo int
	scmap    map[string]scFunc
	block    *StateBlock
	lines    []string
	i        int
	linechan chan *string
}

func newCompiler() *Compiler {
	c := &Compiler{}
	c.scmap = map[string]scFunc{
		"hitby":          c.hitBy,
		"nothitby":       c.notHitBy,
		"assertspecial":  c.assertSpecial,
		"playsnd":        c.playSnd,
		"changestate":    c.changeState,
		"selfstate":      c.selfState,
		"tagin":          c.tagIn,
		"tagout":         c.tagOut,
		"destroyself":    c.destroySelf,
		"changeanim":     c.changeAnim,
		"changeanim2":    c.changeAnim2,
		"helper":         c.helper,
		"ctrlset":        c.ctrlSet,
		"explod":         c.explod,
		"modifyexplod":   c.modifyExplod,
		"gamemakeanim":   c.gameMakeAnim,
		"posset":         c.posSet,
		"posadd":         c.posAdd,
		"velset":         c.velSet,
		"veladd":         c.velAdd,
		"velmul":         c.velMul,
		"palfx":          c.palFX,
		"allpalfx":       c.allPalFX,
		"bgpalfx":        c.bgPalFX,
		"afterimage":     c.afterImage,
		"afterimagetime": c.afterImageTime,
		"hitdef":         c.hitDef,
		"reversaldef":    c.reversalDef,
		"projectile":     c.projectile,
		"width":          c.width,
		"sprpriority":    c.sprPriority,
		"varset":         c.varSet,
		"varadd":         c.varAdd,
		"parentvarset":   c.parentVarSet,
		"parentvaradd":   c.parentVarAdd,
		"turn":           c.turn,
		"targetfacing":   c.targetFacing,
		"targetbind":     c.targetBind,
		"bindtotarget":   c.bindToTarget,
		"targetlifeadd":  c.targetLifeAdd,
		"targetstate":    c.targetState,
		"targetvelset":   c.targetVelSet,
		"targetveladd":   c.targetVelAdd,
		"targetpoweradd": c.targetPowerAdd,
		"targetdrop":     c.targetDrop,
		"lifeadd":        c.lifeAdd,
		"lifeset":        c.lifeSet,
		"poweradd":       c.powerAdd,
		"powerset":       c.powerSet,
		"hitvelset":      c.hitVelSet,
		"screenbound":    c.screenBound,
		"posfreeze":      c.posFreeze,
		"envshake":       c.envShake,
		"hitoverride":    c.hitOverride,
		"pause":          c.pause,
		"superpause":     c.superPause,
	}
	return c
}
func (_ *Compiler) tokenizer(in *string) string {
	*in = strings.TrimSpace(*in)
	if len(*in) == 0 {
		return ""
	}
	switch (*in)[0] {
	case '=':
		*in = (*in)[1:]
		return "="
	case ':':
		if len(*in) >= 2 && (*in)[1] == '=' {
			*in = (*in)[2:]
			return ":="
		}
		*in = (*in)[1:]
		return ":"
	case ';':
		*in = (*in)[1:]
		return ";"
	case '!':
		if len(*in) >= 2 && (*in)[1] == '=' {
			*in = (*in)[2:]
			return "!="
		}
		*in = (*in)[1:]
		return "!"
	case '>':
		if len(*in) >= 2 && (*in)[1] == '=' {
			*in = (*in)[2:]
			return ">="
		}
		*in = (*in)[1:]
		return ">"
	case '<':
		if len(*in) >= 2 && (*in)[1] == '=' {
			*in = (*in)[2:]
			return "<="
		}
		*in = (*in)[1:]
		return "<"
	case '~':
		*in = (*in)[1:]
		return "~"
	case '&':
		if len(*in) >= 2 && (*in)[1] == '&' {
			*in = (*in)[2:]
			return "&&"
		}
		*in = (*in)[1:]
		return "&"
	case '^':
		if len(*in) >= 2 && (*in)[1] == '^' {
			*in = (*in)[2:]
			return "^^"
		}
		*in = (*in)[1:]
		return "^"
	case '|':
		if len(*in) >= 2 && (*in)[1] == '|' {
			*in = (*in)[2:]
			return "||"
		}
		*in = (*in)[1:]
		return "|"
	case '+':
		*in = (*in)[1:]
		return "+"
	case '-':
		*in = (*in)[1:]
		return "-"
	case '*':
		if len(*in) >= 2 && (*in)[1] == '*' {
			*in = (*in)[2:]
			return "**"
		}
		*in = (*in)[1:]
		return "*"
	case '/':
		*in = (*in)[1:]
		return "/"
	case '%':
		*in = (*in)[1:]
		return "%"
	case ',':
		*in = (*in)[1:]
		return ","
	case '(':
		*in = (*in)[1:]
		return "("
	case ')':
		*in = (*in)[1:]
		return ")"
	case '[':
		*in = (*in)[1:]
		return "["
	case ']':
		*in = (*in)[1:]
		return "]"
	case '"':
		*in = (*in)[1:]
		return "\""
	case '{':
		*in = (*in)[1:]
		return "{"
	case '}':
		*in = (*in)[1:]
		return "}"
	}
	i, ten := 0, false
	for ; i < len(*in); i++ {
		if (*in)[i] == '.' {
			if ten {
				break
			}
			ten = true
		} else if (*in)[i] < '0' || (*in)[i] > '9' {
			break
		}
	}
	if i > 0 && i < len(*in) && ((*in)[i] == 'e' || (*in)[i] == 'E') {
		j := i + 1
		for i++; i < len(*in); i++ {
			if ((*in)[i] < '0' || (*in)[i] > '9') &&
				(i != j || ((*in)[i] != '-' && (*in)[i] != '+')) {
				break
			}
		}
	}
	if i == 0 {
		i = strings.IndexAny(*in, kuuhaktokigou)
		if i < 0 {
			i = len(*in)
		}
	}
	token := strings.ToLower((*in)[:i])
	*in = (*in)[i:]
	return token
}
func (_ *Compiler) isOperator(token string) int {
	switch token {
	case "", ",", ")", "]":
		return -1
	case "||":
		return 1
	case "^^":
		return 2
	case "&&":
		return 3
	case "|":
		return 4
	case "^":
		return 5
	case "&":
		return 6
	case "=", "!=":
		return 7
	case ">", ">=", "<", "<=":
		return 8
	case "+", "-":
		return 9
	case "*", "/", "%":
		return 10
	case "**":
		return 11
	}
	return 0
}
func (c *Compiler) operator(in *string) error {
	if len(c.maeOp) > 0 {
		if opp := c.isOperator(c.token); opp <= c.isOperator(c.maeOp) {
			if opp < 0 || ((!c.usiroOp || c.token[0] != '(') &&
				(c.token[0] < 'A' || c.token[0] > 'Z') &&
				(c.token[0] < 'a' || c.token[0] > 'z')) {
				return Error(c.maeOp + "が不正です")
			}
			*in = c.token + " " + *in
			c.token = c.maeOp
			c.maeOp = ""
			c.norange = true
		}
	}
	return nil
}
func (c *Compiler) integer2(in *string) (int32, error) {
	istr := c.token
	c.token = c.tokenizer(in)
	minus := istr == "-"
	if minus {
		istr = c.token
		c.token = c.tokenizer(in)
	}
	for _, c := range istr {
		if c < '0' || c > '9' {
			return 0, Error(istr + "が整数でありません")
		}
	}
	i := Atoi(istr)
	if minus {
		i *= -1
	}
	return i, nil
}
func (c *Compiler) number(token string) BytecodeValue {
	f, err := strconv.ParseFloat(token, 64)
	if err != nil && f == 0 {
		return bvNone()
	}
	if strings.Index(token, ".") >= 0 {
		c.usiroOp = false
		return BytecodeValue{VT_Float, f}
	}
	if strings.IndexAny(token, "Ee") >= 0 {
		return bvNone()
	}
	c.usiroOp = false
	if f > math.MaxInt32 {
		return BytecodeValue{VT_Int, float64(math.MaxInt32)}
	}
	if f < math.MinInt32 {
		return BytecodeValue{VT_Int, float64(math.MinInt32)}
	}
	return BytecodeValue{VT_Int, f}
}
func (c *Compiler) attr(text string, hitdef bool) (int32, error) {
	flg := int32(0)
	att := SplitAndTrim(text, ",")
	for _, a := range att[0] {
		switch a {
		case 'S', 's':
			if hitdef {
				flg = int32(ST_S)
			} else {
				flg |= int32(ST_S)
			}
		case 'C', 'c':
			if hitdef {
				flg = int32(ST_C)
			} else {
				flg |= int32(ST_C)
			}
		case 'A', 'a':
			if hitdef {
				flg = int32(ST_A)
			} else {
				flg |= int32(ST_A)
			}
		default:
			if sys.ignoreMostErrors && a < 128 && (a < 'A' || a > 'Z') &&
				(a < 'a' || a > 'z') {
				return flg, nil
			}
			return 0, Error(string(a) + "が無効な値です")
		}
	}
	hitdefflg := flg
	for i, a := range att[1:] {
		l := len(a)
		if sys.ignoreMostErrors && l >= 2 {
			a = strings.TrimSpace(a[:2])
		}
		switch strings.ToLower(a) {
		case "na":
			flg |= int32(AT_NA)
		case "nt":
			flg |= int32(AT_NT)
		case "np":
			flg |= int32(AT_NP)
		case "sa":
			flg |= int32(AT_SA)
		case "st":
			flg |= int32(AT_ST)
		case "sp":
			flg |= int32(AT_SP)
		case "ha":
			flg |= int32(AT_HA)
		case "ht":
			flg |= int32(AT_HT)
		case "hp":
			flg |= int32(AT_HP)
		case "aa":
			flg |= int32(AT_AA)
		case "at":
			flg |= int32(AT_AT)
		case "ap":
			flg |= int32(AT_AP)
		case "n":
			flg |= int32(AT_NA | AT_NT | AT_NP)
		case "s":
			flg |= int32(AT_SA | AT_ST | AT_SP)
		case "h", "a":
			flg |= int32(AT_HA | AT_HT | AT_HP)
		default:
			if sys.ignoreMostErrors && sys.cgi[c.playerNo].ver[0] == 1 {
				if hitdef {
					flg = hitdefflg
				}
				return flg, nil
			}
			return 0, Error(a + "が無効な値です")
		}
		if i == 0 {
			hitdefflg = flg
		}
		if l > 2 {
			break
		}
	}
	if hitdef {
		flg = hitdefflg
	}
	return flg, nil
}
func (c *Compiler) kakkohiraku(in *string) error {
	if c.tokenizer(in) != "(" {
		return Error(c.token + "の次に'('がありません")
	}
	c.token = c.tokenizer(in)
	return nil
}
func (c *Compiler) kakkotojiru(in *string) error {
	if c.token != ")" {
		return Error(c.token + "の前に')'がありません")
	}
	return nil
}
func (c *Compiler) kyuushiki(in *string) (not bool, err error) {
	for {
		c.token = c.tokenizer(in)
		if len(c.token) > 0 {
			if c.token == "!=" {
				not = true
				break
			} else if c.token == "=" {
				break
			} else if sys.ignoreMostErrors {
				if c.token[len(c.token)-1] == '=' {
					break
				}
				continue
			}
		}
		return false, Error("'='か'!='がありません")
	}
	c.token = c.tokenizer(in)
	return
}
func (c *Compiler) intRange(in *string) (minop OpCode, maxop OpCode,
	min, max int32, err error) {
	switch c.token {
	case "(":
		minop = OC_gt
	case "[":
		minop = OC_ge
	default:
		err = Error("'['か'('がありません")
		return
	}
	var intf func(in *string) (int32, error)
	if sys.ignoreMostErrors {
		intf = func(in *string) (int32, error) {
			c.token = c.tokenizer(in)
			minus := false
			for c.token == "-" || c.token == "+" {
				minus = minus || c.token == "-"
				c.token = c.tokenizer(in)
			}
			if len(c.token) == 0 || c.token[0] < '0' || c.token[0] > '9' {
				return 0, Error("数字の読み込みエラーです")
			}
			i := Atoi(c.token)
			if minus {
				i *= -1
			}
			return i, nil
		}
	} else {
		intf = c.integer2
	}
	if min, err = intf(in); err != nil {
		return
	}
	if sys.ignoreMostErrors {
		if i := strings.Index(*in, ","); i >= 0 {
			c.token = ","
			*in = (*in)[i+1:]
		}
	} else {
		c.token = c.tokenizer(in)
	}
	if c.token != "," {
		err = Error("','がありません")
		return
	}
	if max, err = intf(in); err != nil {
		return
	}
	if sys.ignoreMostErrors {
		if i := strings.IndexAny(*in, "])"); i >= 0 {
			c.token = string((*in)[i])
			*in = (*in)[i+1:]
		}
	} else {
		c.token = c.tokenizer(in)
	}
	switch c.token {
	case ")":
		maxop = OC_lt
	case "]":
		maxop = OC_le
	default:
		err = Error("']'か')'がありません")
		return
	}
	c.token = c.tokenizer(in)
	return
}
func (c *Compiler) kyuushikiThroughNeo(_range bool, in *string) {
	if sys.ignoreMostErrors {
		i := 0
		for ; i < len(*in); i++ {
			if (*in)[i] >= '0' && (*in)[i] <= '9' || (*in)[i] == '-' ||
				_range && ((*in)[i] == '[' || (*in)[i] == '(') {
				break
			}
		}
		*in = (*in)[i:]
	}
	c.token = c.tokenizer(in)
}
func (c *Compiler) kyuushikiSuperDX(out *BytecodeExp, in *string,
	hissu bool) error {
	comma := c.token == ","
	if comma {
		c.token = c.tokenizer(in)
	}
	var opc OpCode
	switch c.token {
	case "<":
		opc = OC_lt
		c.kyuushikiThroughNeo(false, in)
	case ">":
		opc = OC_gt
		c.kyuushikiThroughNeo(false, in)
	case "<=":
		opc = OC_le
		c.kyuushikiThroughNeo(false, in)
	case ">=":
		opc = OC_ge
		c.kyuushikiThroughNeo(false, in)
	default:
		opc = OC_eq
		hikaku := false
		switch c.token {
		case "!=":
			opc = OC_ne
			hikaku = true
		case "=":
			hikaku = true
		default:
			if hissu && !comma {
				return Error("比較演算子がありません")
			}
		}
		if hikaku {
			c.kyuushikiThroughNeo(true, in)
		}
		if c.token == "[" || c.token == "(" {
			minop, maxop, min, max, err := c.intRange(in)
			if err != nil {
				return err
			}
			if opc == OC_ne {
				if minop == OC_gt {
					minop = OC_le
				} else {
					minop = OC_lt
				}
				if maxop == OC_lt {
					minop = OC_ge
				} else {
					minop = OC_gt
				}
			}
			out.append(OC_dup)
			out.appendValue(BytecodeInt(min))
			out.append(minop)
			out.append(OC_swap)
			out.appendValue(BytecodeInt(max))
			out.append(maxop)
			if opc == OC_ne {
				out.append(OC_blor)
			} else {
				out.append(OC_bland)
			}
			c.usiroOp = comma || hikaku
			return nil
		}
	}
	n, err := c.integer2(in)
	if err != nil {
		return err
	}
	out.appendValue(BytecodeInt(n))
	out.append(opc)
	c.usiroOp = true
	return nil
}
func (c *Compiler) oneArg(out *BytecodeExp, in *string,
	rd, appendVal bool) (BytecodeValue, error) {
	if err := c.kakkohiraku(in); err != nil {
		return bvNone(), err
	}
	var be BytecodeExp
	bv, err := c.expBoolOr(&be, in)
	if err != nil {
		return bvNone(), err
	}
	if err := c.kakkotojiru(in); err != nil {
		return bvNone(), err
	}
	if appendVal {
		be.appendValue(bv)
		bv = bvNone()
	}
	if rd && len(be) > 0 {
		out.appendI32Op(OC_nordrun, int32(len(be)))
	}
	out.append(be...)
	return bv, nil
}
func (c *Compiler) mathFunc(out *BytecodeExp, in *string, rd bool,
	oc OpCode, f func(*BytecodeValue)) (bv BytecodeValue, err error) {
	var be BytecodeExp
	if bv, err = c.oneArg(&be, in, false, false); err != nil {
		return
	}
	if bv.IsNone() {
		if rd {
			out.append(OC_rdreset)
		}
		out.append(be...)
		out.append(oc)
	} else {
		f(&bv)
	}
	return
}
func (c *Compiler) expValue(out *BytecodeExp, in *string,
	rd bool) (BytecodeValue, error) {
	c.usiroOp, c.norange = true, false
	bv := c.number(c.token)
	if !bv.IsNone() {
		c.token = c.tokenizer(in)
		return bv, nil
	}
	if !sys.ignoreMostErrors {
		defer func() { c.usiroOp = false }()
	}
	_var := func(sys, f bool) error {
		bv1, err := c.oneArg(out, in, rd, false)
		if err != nil {
			return err
		}
		var oc OpCode
		set, _else := c.token == ":=", false
		if !bv1.IsNone() && bv1.ToI() >= 0 {
			switch [2]bool{sys, f} {
			case [2]bool{false, false}:
				if bv1.ToI() < int32(NumVar) {
					oc = OC_var0 + OpCode(bv1.ToI()) // OC_st_var0と同じ値
				} else {
					_else = true
				}
			case [2]bool{false, true}:
				if bv1.ToI() < int32(NumFvar) {
					oc = OC_fvar0 + OpCode(bv1.ToI()) // OC_st_fvar0と同じ値
				} else {
					_else = true
				}
			case [2]bool{true, false}:
				if bv1.ToI() < int32(NumSysVar) {
					oc = OC_sysvar0 + OpCode(bv1.ToI()) // OC_st_sysvar0と同じ値
				} else {
					_else = true
				}
			case [2]bool{true, true}:
				if bv1.ToI() < int32(NumSysFvar) {
					oc = OC_sysfvar0 + OpCode(bv1.ToI()) // OC_st_sysfvar0と同じ値
				} else {
					_else = true
				}
			}
		} else {
			_else = true
		}
		if set {
			c.token = c.tokenizer(in)
			var be2 BytecodeExp
			bv2, err := c.expEqne(&be2, in)
			if err != nil {
				return err
			}
			be2.appendValue(bv2)
			if rd {
				out.appendI32Op(OC_nordrun, int32(len(be2)))
			}
			out.append(be2...)
			out.append(OC_st_)
		}
		if _else {
			switch [2]bool{sys, f} {
			case [2]bool{false, false}:
				oc = OC_var
			case [2]bool{false, true}:
				oc = OC_fvar
			case [2]bool{true, false}:
				oc = OC_sysvar
			case [2]bool{true, true}:
				oc = OC_sysfvar
			}
			if set {
				oc += OC_st_var - OC_var
			}
		}
		out.append(oc)
		return nil
	}
	text := func() error {
		i := strings.Index(*in, "\"")
		if c.token != "\"" || i < 0 {
			return Error("\"で囲まれていません")
		}
		c.token = (*in)[:i]
		*in = (*in)[i+1:]
		return nil
	}
	eqne := func(f func() error) error {
		not, err := c.kyuushiki(in)
		if err != nil {
			return err
		}
		if err := f(); err != nil {
			return err
		}
		if not {
			out.append(OC_blnot)
		}
		return nil
	}
	var be1, be2, be3 BytecodeExp
	var bv1, bv2, bv3 BytecodeValue
	var n int32
	var be BytecodeExp
	var opc OpCode
	var err error
	switch c.token {
	case "":
		return bvNone(), Error("空です")
	case "root", "parent", "helper", "target", "partner",
		"enemy", "enemynear", "playerid":
		switch c.token {
		case "parent":
			opc = OC_parent
			c.token = c.tokenizer(in)
		case "root":
			opc = OC_root
			c.token = c.tokenizer(in)
		default:
			switch c.token {
			case "helper":
				opc = OC_helper
			case "target":
				opc = OC_target
			case "partner":
				opc = OC_partner
			case "enemy":
				opc = OC_enemy
			case "enemynear":
				opc = OC_enemynear
			case "playerid":
				opc = OC_playerid
			}
			c.token = c.tokenizer(in)
			if c.token == "(" {
				c.token = c.tokenizer(in)
				if bv1, err = c.expBoolOr(&be1, in); err != nil {
					return bvNone(), err
				}
				if err := c.kakkotojiru(in); err != nil {
					return bvNone(), err
				}
				c.token = c.tokenizer(in)
				be1.appendValue(bv1)
			} else {
				switch opc {
				case OC_helper, OC_target:
					be1.appendValue(BytecodeInt(-1))
				case OC_partner, OC_enemy, OC_enemynear:
					be1.appendValue(BytecodeInt(0))
				case OC_playerid:
					return bvNone(), Error("playeridの次に'('がありません")
				}
			}
		}
		if rd {
			out.appendI32Op(OC_nordrun, int32(len(be1)))
		}
		out.append(be1...)
		if c.token != "," {
			return bvNone(), Error(",がありません")
		}
		c.token = c.tokenizer(in)
		if bv2, err = c.expValue(&be2, in, true); err != nil {
			return bvNone(), err
		}
		be2.appendValue(bv2)
		out.appendI32Op(opc, int32(len(be2)))
		out.append(be2...)
		return bvNone(), nil
	case "-":
		if len(*in) > 0 && (((*in)[0] >= '0' && (*in)[0] <= '9') || (*in)[0] == '.') {
			c.token += c.tokenizer(in)
			bv = c.number(c.token)
			if bv.IsNone() {
				return bvNone(), Error(c.token + "が不正です")
			}
		} else {
			c.token = c.tokenizer(in)
			if bv, err = c.expValue(&be1, in, false); err != nil {
				return bvNone(), err
			}
			if bv.IsNone() {
				if rd {
					out.append(OC_rdreset)
				}
				out.append(be1...)
				out.append(OC_neg)
			} else {
				out.neg(&bv)
			}
			return bv, nil
		}
	case "~":
		c.token = c.tokenizer(in)
		if bv, err = c.expValue(&be1, in, false); err != nil {
			return bvNone(), err
		}
		if bv.IsNone() {
			if rd {
				out.append(OC_rdreset)
			}
			out.append(be1...)
			out.append(OC_not)
		} else {
			out.not(&bv)
		}
		return bv, nil
	case "!":
		c.token = c.tokenizer(in)
		if bv, err = c.expValue(&be1, in, false); err != nil {
			return bvNone(), err
		}
		if bv.IsNone() {
			if rd {
				out.append(OC_rdreset)
			}
			out.append(be1...)
			out.append(OC_blnot)
		} else {
			out.blnot(&bv)
		}
		return bv, nil
	case "(":
		c.token = c.tokenizer(in)
		if bv, err = c.expBoolOr(&be1, in); err != nil {
			return bvNone(), err
		}
		if bv.IsNone() {
			if rd {
				out.append(OC_rdreset)
			}
			out.append(be1...)
		}
		if err := c.kakkotojiru(in); err != nil {
			return bvNone(), err
		}
	case "var":
		if err := _var(false, false); err != nil {
			return bvNone(), err
		}
	case "fvar":
		if err := _var(false, true); err != nil {
			return bvNone(), err
		}
	case "sysvar":
		if err := _var(true, false); err != nil {
			return bvNone(), err
		}
	case "sysfvar":
		if err := _var(true, true); err != nil {
			return bvNone(), err
		}
	case "ifelse", "cond":
		cond := c.token == "cond"
		if err := c.kakkohiraku(in); err != nil {
			return bvNone(), err
		}
		if bv1, err = c.expBoolOr(&be1, in); err != nil {
			return bvNone(), err
		}
		if c.token != "," {
			return bvNone(), Error("','がありません")
		}
		c.token = c.tokenizer(in)
		if bv2, err = c.expBoolOr(&be2, in); err != nil {
			return bvNone(), err
		}
		if c.token != "," {
			return bvNone(), Error("','がありません")
		}
		c.token = c.tokenizer(in)
		if bv3, err = c.expBoolOr(&be3, in); err != nil {
			return bvNone(), err
		}
		if err := c.kakkotojiru(in); err != nil {
			return bvNone(), err
		}
		if bv1.IsNone() || bv2.IsNone() || bv3.IsNone() {
			if cond {
				be3.appendValue(bv3)
				be2.appendValue(bv2)
				if len(be3) > int(math.MaxUint8-1) {
					be2.appendI32Op(OC_jmp, int32(len(be3)+1))
				} else {
					be2.append(OC_jmp8, OpCode(len(be3)+1))
				}
				be1.appendValue(bv1)
				if len(be2) > int(math.MaxUint8-1) {
					be1.appendI32Op(OC_jz, int32(len(be2)+1))
				} else {
					be1.append(OC_jz8, OpCode(len(be2)+1))
				}
				be1.append(OC_pop)
				be1.append(be2...)
				be1.append(OC_pop)
				be1.append(be3...)
				if rd {
					out.appendI32Op(OC_run, int32(len(be1)))
				}
				out.append(be1...)
			} else {
				if rd {
					out.append(OC_rdreset)
				}
				out.append(be1...)
				out.appendValue(bv1)
				out.append(be2...)
				out.appendValue(bv2)
				out.append(be3...)
				out.appendValue(bv3)
				out.append(OC_ifelse)
			}
		} else {
			if bv1.ToB() {
				bv = bv2
			} else {
				bv = bv3
			}
		}
	case "time":
		out.append(OC_time)
	case "alive":
		out.append(OC_alive)
	case "random":
		out.append(OC_random)
	case "roundstate":
		out.append(OC_roundstate)
	case "stateno":
		out.append(OC_stateno)
	case "prevstateno":
		out.append(OC_prevstateno)
	case "p2stateno":
		out.appendI32Op(OC_p2, 1)
		out.append(OC_stateno)
	case "movecontact":
		out.append(OC_movecontact)
	case "movehit":
		out.append(OC_movehit)
	case "moveguarded":
		out.append(OC_moveguarded)
	case "movereversed":
		out.append(OC_movereversed)
	case "canrecover":
		out.append(OC_canrecover)
	case "hitshakeover":
		out.append(OC_hitshakeover)
	case "matchover":
		out.append(OC_matchover)
	case "anim":
		out.append(OC_anim)
	case "animtime":
		out.append(OC_animtime)
	case "animelem":
		if _, err = c.kyuushiki(in); err != nil {
			return bvNone(), err
		}
		if c.token == "-" {
			return bvNone(), Error("マイナスが付くとエラーです")
		}
		if n, err = c.integer2(in); err != nil {
			return bvNone(), err
		}
		if n <= 0 {
			return bvNone(), Error("animelemのは0より大きくなければいけません")
		}
		be1.appendValue(BytecodeInt(n))
		if rd {
			out.appendI32Op(OC_nordrun, int32(len(be1)))
		}
		out.append(be1...)
		out.append(OC_animelemtime)
		if err = c.kyuushikiSuperDX(&be, in, false); err != nil {
			return bvNone(), err
		}
		out.append(OC_jsf8, OpCode(len(be)))
		out.append(be...)
	case "animelemtime":
		if _, err := c.oneArg(out, in, rd, true); err != nil {
			return bvNone(), err
		}
		out.append(OC_animelemtime)
	case "animexist":
		if _, err := c.oneArg(out, in, rd, true); err != nil {
			return bvNone(), err
		}
		out.append(OC_animexist)
	case "selfanimexist":
		if _, err := c.oneArg(out, in, rd, true); err != nil {
			return bvNone(), err
		}
		out.append(OC_selfanimexist)
	case "vel":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_vel_x)
		case "y":
			out.append(OC_vel_y)
		case "z":
			bv = BytecodeFloat(0)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "pos":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_pos_x)
		case "y":
			out.append(OC_pos_y)
		case "z":
			bv = BytecodeFloat(0)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "screenpos":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_screenpos_x)
		case "y":
			out.append(OC_screenpos_y)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "command":
		if err := eqne(func() error {
			if err := text(); err != nil {
				return err
			}
			i, ok := c.cmdl.Names[c.token]
			if !ok {
				return Error("コマンド\"" + c.token + "\"は存在しません")
			}
			out.appendI32Op(OC_command, int32(i))
			return nil
		}); err != nil {
			return bvNone(), err
		}
	case "p2dist":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_ex_, OC_ex_p2dist_x)
		case "y":
			out.append(OC_ex_, OC_ex_p2dist_y)
		case "z":
			bv = BytecodeFloat(0)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "p2bodydist":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_ex_, OC_ex_p2bodydist_x)
		case "y":
			out.append(OC_ex_, OC_ex_p2bodydist_y)
		case "z":
			bv = BytecodeFloat(0)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "rootdist":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_ex_, OC_ex_rootdist_x)
		case "y":
			out.append(OC_ex_, OC_ex_rootdist_y)
		case "z":
			bv = BytecodeFloat(0)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "parentdist":
		c.token = c.tokenizer(in)
		switch c.token {
		case "x":
			out.append(OC_ex_, OC_ex_parentdist_x)
		case "y":
			out.append(OC_ex_, OC_ex_parentdist_y)
		case "z":
			bv = BytecodeFloat(0)
		default:
			return bvNone(), Error(c.token + "が不正です")
		}
	case "frontedgedist":
		out.append(OC_frontedgedist)
	case "frontedgebodydist":
		out.append(OC_frontedgebodydist)
	case "frontedge":
		out.append(OC_frontedge)
	case "backedgedist":
		out.append(OC_backedgedist)
	case "backedgebodydist":
		out.append(OC_backedgebodydist)
	case "backedge":
		out.append(OC_backedge)
	case "leftedge":
		out.append(OC_leftedge)
	case "rightedge":
		out.append(OC_rightedge)
	case "topedge":
		out.append(OC_topedge)
	case "bottomedge":
		out.append(OC_bottomedge)
	case "abs":
		if bv, err = c.mathFunc(out, in, rd, OC_abs, out.abs); err != nil {
			return bvNone(), err
		}
	case "exp":
		if bv, err = c.mathFunc(out, in, rd, OC_exp, out.exp); err != nil {
			return bvNone(), err
		}
	case "ln":
		if bv, err = c.mathFunc(out, in, rd, OC_ln, out.ln); err != nil {
			return bvNone(), err
		}
	case "log":
		if err := c.kakkohiraku(in); err != nil {
			return bvNone(), err
		}
		if bv1, err = c.expBoolOr(&be1, in); err != nil {
			return bvNone(), err
		}
		if c.token != "," {
			return bvNone(), Error("','がありません")
		}
		c.token = c.tokenizer(in)
		if bv2, err = c.expBoolOr(&be2, in); err != nil {
			return bvNone(), err
		}
		if err := c.kakkotojiru(in); err != nil {
			return bvNone(), err
		}
		if bv1.IsNone() || bv2.IsNone() {
			if rd {
				out.append(OC_rdreset)
			}
			out.append(be1...)
			out.append(be2...)
			out.append(OC_log)
			bv = bvNone()
		} else {
			out.log(&bv1, bv2)
			bv = bv1
		}
	case "cos":
		if bv, err = c.mathFunc(out, in, rd, OC_cos, out.cos); err != nil {
			return bvNone(), err
		}
	case "sin":
		if bv, err = c.mathFunc(out, in, rd, OC_sin, out.sin); err != nil {
			return bvNone(), err
		}
	case "tan":
		if bv, err = c.mathFunc(out, in, rd, OC_tan, out.tan); err != nil {
			return bvNone(), err
		}
	case "acos":
		if bv, err = c.mathFunc(out, in, rd, OC_acos, out.acos); err != nil {
			return bvNone(), err
		}
	case "asin":
		if bv, err = c.mathFunc(out, in, rd, OC_asin, out.asin); err != nil {
			return bvNone(), err
		}
	case "atan":
		if bv, err = c.mathFunc(out, in, rd, OC_atan, out.atan); err != nil {
			return bvNone(), err
		}
	case "floor":
		if bv, err = c.mathFunc(out, in, rd, OC_floor, out.floor); err != nil {
			return bvNone(), err
		}
	case "ceil":
		if bv, err = c.mathFunc(out, in, rd, OC_ceil, out.ceil); err != nil {
			return bvNone(), err
		}
	case "gethitvar":
		if err := c.kakkohiraku(in); err != nil {
			return bvNone(), err
		}
		switch c.token {
		case "xveladd":
			bv.SetF(0)
		case "yveladd":
			bv.SetF(0)
		case "type":
			bv.SetI(0)
		case "zoff":
			bv.SetF(0)
		case "fall.envshake.dir":
			bv.SetI(0)
		default:
			out.append(OC_ex_)
			switch c.token {
			case "animtype":
				out.append(OC_ex_gethitvar_animtype)
			case "airtype":
				out.append(OC_ex_gethitvar_airtype)
			case "groundtype":
				out.append(OC_ex_gethitvar_groundtype)
			case "damage":
				out.append(OC_ex_gethitvar_damage)
			case "hitcount":
				out.append(OC_ex_gethitvar_hitcount)
			case "fallcount":
				out.append(OC_ex_gethitvar_fallcount)
			case "hitshaketime":
				out.append(OC_ex_gethitvar_hitshaketime)
			case "hittime":
				out.append(OC_ex_gethitvar_hittime)
			case "slidetime":
				out.append(OC_ex_gethitvar_slidetime)
			case "ctrltime":
				out.append(OC_ex_gethitvar_ctrltime)
			case "recovertime":
				out.append(OC_ex_gethitvar_recovertime)
			case "xoff":
				out.append(OC_ex_gethitvar_xoff)
			case "yoff":
				out.append(OC_ex_gethitvar_yoff)
			case "xvel":
				out.append(OC_ex_gethitvar_xvel)
			case "yvel":
				out.append(OC_ex_gethitvar_yvel)
			case "yaccel":
				out.append(OC_ex_gethitvar_yaccel)
			case "hitid", "chainid":
				out.append(OC_ex_gethitvar_chainid)
			case "guarded":
				out.append(OC_ex_gethitvar_guarded)
			case "isbound":
				out.append(OC_ex_gethitvar_isbound)
			case "fall":
				out.append(OC_ex_gethitvar_fall)
			case "fall.damage":
				out.append(OC_ex_gethitvar_fall_damage)
			case "fall.xvel":
				out.append(OC_ex_gethitvar_fall_xvel)
			case "fall.yvel":
				out.append(OC_ex_gethitvar_fall_yvel)
			case "fall.recover":
				out.append(OC_ex_gethitvar_fall_recover)
			case "fall.time":
				out.append(OC_ex_gethitvar_fall_time)
			case "fall.recovertime":
				out.append(OC_ex_gethitvar_fall_recovertime)
			case "fall.kill":
				out.append(OC_ex_gethitvar_fall_kill)
			case "fall.envshake.time":
				out.append(OC_ex_gethitvar_fall_envshake_time)
			case "fall.envshake.freq":
				out.append(OC_ex_gethitvar_fall_envshake_freq)
			case "fall.envshake.ampl":
				out.append(OC_ex_gethitvar_fall_envshake_ampl)
			case "fall.envshake.phase":
				out.append(OC_ex_gethitvar_fall_envshake_phase)
			default:
				return bvNone(), Error(c.token + "が不正です")
			}
		}
		c.token = c.tokenizer(in)
		if err := c.kakkotojiru(in); err != nil {
			return bvNone(), err
		}
	default:
		println(c.token)
		unimplemented()
	}
	c.token = c.tokenizer(in)
	return bv, nil
}
func (c *Compiler) renzokuEnzansihaError(in *string) error {
	*in = strings.TrimSpace(*in)
	if len(*in) > 0 {
		switch (*in)[0] {
		default:
			if len(*in) < 2 || (*in)[:2] != "!=" {
				break
			}
			fallthrough
		case '=', '<', '>', '|', '&', '+', '*', '/', '%', '^':
			return Error(c.tokenizer(in) + "が不正です")
		}
	}
	return nil
}
func (c *Compiler) expPostNot(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	bv, err := c.expValue(out, in, false)
	if err != nil {
		return bvNone(), err
	}
	for c.token == "!" {
		c.usiroOp = true
		if bv.IsNone() {
			out.append(OC_blnot)
		} else {
			out.blnot(&bv)
		}
		c.token = c.tokenizer(in)
	}
	if len(c.maeOp) == 0 {
		opp := c.isOperator(c.token)
		if opp == 0 {
			if !c.usiroOp && c.token == "(" {
				return bvNone(), Error("演算子がありません")
			}
			oldtoken, oldin := c.token, *in
			var dummyout BytecodeExp
			if _, err := c.expValue(&dummyout, in, false); err != nil {
				return bvNone(), err
			}
			if c.isOperator(c.token) <= 0 {
				return bvNone(), Error("演算子がありません")
			}
			if err := c.renzokuEnzansihaError(in); err != nil {
				return bvNone(), err
			}
			oldin = oldin[:len(oldin)-len(*in)]
			*in = oldtoken + " " + oldin[:strings.LastIndex(oldin, c.token)] + " " +
				*in
		} else if opp > 0 {
			if err := c.renzokuEnzansihaError(in); err != nil {
				return bvNone(), err
			}
		}
	}
	return bv, nil
}
func (c *Compiler) expPow(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	bv, err := c.expPostNot(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		if c.token == "**" {
			c.token = c.tokenizer(in)
			var be BytecodeExp
			bv2, err := c.expPostNot(&be, in)
			if err != nil {
				return bvNone(), err
			}
			if bv.IsNone() || bv2.IsNone() {
				out.appendValue(bv)
				out.append(be...)
				out.appendValue(bv2)
				out.append(OC_pow)
				bv = bvNone()
			} else {
				out.pow(&bv, bv2, c.playerNo)
			}
		} else {
			break
		}
	}
	return bv, nil
}
func (c *Compiler) expMldv(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	bv, err := c.expPow(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		switch c.token {
		case "*":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expPow, out.mul, OC_mul)
		case "/":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expPow, out.div, OC_div)
		case "%":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expPow, out.mod, OC_mod)
		default:
			return bv, nil
		}
		if err != nil {
			return bvNone(), err
		}
	}
}
func (c *Compiler) expAdsb(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	bv, err := c.expMldv(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		switch c.token {
		case "+":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expMldv, out.add, OC_add)
		case "-":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expMldv, out.sub, OC_sub)
		default:
			return bv, nil
		}
		if err != nil {
			return bvNone(), err
		}
	}
}
func (c *Compiler) expGrls(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	bv, err := c.expAdsb(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		switch c.token {
		case ">":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expAdsb, out.gt, OC_gt)
		case ">=":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expAdsb, out.ge, OC_ge)
		case "<":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expAdsb, out.lt, OC_lt)
		case "<=":
			c.token = c.tokenizer(in)
			err = c.expOneOpSub(out, in, &bv, c.expAdsb, out.le, OC_le)
		default:
			return bv, nil
		}
		if err != nil {
			return bvNone(), err
		}
	}
}
func (c *Compiler) expRange(out *BytecodeExp, in *string,
	bv *BytecodeValue, opc OpCode) error {
	open := c.token
	c.token = c.tokenizer(in)
	var be2, be3 BytecodeExp
	bv2, err := c.expBoolOr(&be2, in)
	if err != nil {
		return err
	}
	if c.token != "," {
		if open != "(" {
			return Error(",がありません")
		}
		if err := c.kakkotojiru(in); err != nil {
			return err
		}
		c.token = c.tokenizer(in)
		if bv.IsNone() || bv2.IsNone() {
			out.appendValue(*bv)
			out.append(be2...)
			out.appendValue(bv2)
			out.append(opc)
			*bv = bvNone()
		} else {
			switch opc {
			case OC_eq:
				out.eq(bv, bv2)
			case OC_ne:
				out.ne(bv, bv2)
			}
		}
		return nil
	}
	c.token = c.tokenizer(in)
	bv3, err := c.expBoolOr(&be3, in)
	close := c.token
	if close != "]" && close != ")" {
		return Error("]か)がありません")
	}
	c.token = c.tokenizer(in)
	if bv.IsNone() || bv2.IsNone() || bv3.IsNone() {
		var op1, op2, op3 OpCode
		if opc == OC_ne {
			if open == "(" {
				op1 = OC_le
			} else {
				op1 = OC_lt
			}
			if close == ")" {
				op2 = OC_ge
			} else {
				op2 = OC_gt
			}
			op3 = OC_blor
		} else {
			if open == "(" {
				op1 = OC_gt
			} else {
				op1 = OC_ge
			}
			if close == ")" {
				op2 = OC_lt
			} else {
				op2 = OC_le
			}
			op3 = OC_bland
		}
		out.appendValue(*bv)
		out.append(OC_dup)
		out.append(be2...)
		out.appendValue(bv2)
		out.append(op1)
		out.append(OC_swap)
		out.append(be3...)
		out.appendValue(bv3)
		out.append(op2)
		out.append(op3)
		*bv = bvNone()
	} else {
		tmp := *bv
		if open == "(" {
			out.gt(&tmp, bv2)
		} else {
			out.ge(&tmp, bv2)
		}
		if close == ")" {
			out.lt(bv, bv3)
		} else {
			out.le(bv, bv3)
		}
		bv.SetB(tmp.ToB() && bv.ToB())
		if opc == OC_ne {
			bv.SetB(!bv.ToB())
		}
	}
	return nil
}
func (c *Compiler) expEqne(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	bv, err := c.expGrls(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		var opc OpCode
		switch c.token {
		case "=":
			opc = OC_eq
		case "!=":
			opc = OC_ne
		default:
			return bv, nil
		}
		c.token = c.tokenizer(in)
		switch c.token {
		case "[", "(":
			if !c.norange {
				if err = c.expRange(out, in, &bv, opc); err != nil {
					return bvNone(), err
				}
				break
			}
			fallthrough
		default:
			switch opc {
			case OC_eq:
				err = c.expOneOpSub(out, in, &bv, c.expGrls, out.eq, opc)
			case OC_ne:
				err = c.expOneOpSub(out, in, &bv, c.expGrls, out.ne, opc)
			}
			if err != nil {
				return bvNone(), err
			}
		}
	}
}
func (_ *Compiler) expOneOpSub(out *BytecodeExp, in *string, bv *BytecodeValue,
	ef expFunc, opf func(v1 *BytecodeValue, v2 BytecodeValue),
	opc OpCode) error {
	var be BytecodeExp
	bv2, err := ef(&be, in)
	if err != nil {
		return err
	}
	if bv.IsNone() || bv2.IsNone() {
		out.appendValue(*bv)
		out.append(be...)
		out.appendValue(bv2)
		out.append(opc)
		*bv = bvNone()
	} else {
		opf(bv, bv2)
	}
	return nil
}
func (c *Compiler) expOneOp(out *BytecodeExp, in *string, ef expFunc,
	opt string, opf func(v1 *BytecodeValue, v2 BytecodeValue),
	opc OpCode) (BytecodeValue, error) {
	bv, err := ef(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		if c.token == opt {
			c.token = c.tokenizer(in)
			if err := c.expOneOpSub(out, in, &bv, ef, opf, opc); err != nil {
				return bvNone(), err
			}
		} else {
			return bv, nil
		}
	}
}
func (c *Compiler) expAnd(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	return c.expOneOp(out, in, c.expEqne, "&", out.and, OC_and)
}
func (c *Compiler) expXor(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	return c.expOneOp(out, in, c.expAnd, "^", out.xor, OC_xor)
}
func (c *Compiler) expOr(out *BytecodeExp, in *string) (BytecodeValue, error) {
	return c.expOneOp(out, in, c.expXor, "|", out.or, OC_or)
}
func (c *Compiler) expBoolAnd(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	if c.block != nil {
		return c.expOneOp(out, in, c.expOr, "&&", out.bland, OC_bland)
	}
	bv, err := c.expOr(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		if c.token == "&&" {
			c.token = c.tokenizer(in)
			var be BytecodeExp
			bv2, err := c.expOr(&be, in)
			if err != nil {
				return bvNone(), err
			}
			if bv.IsNone() || bv2.IsNone() {
				out.appendValue(bv)
				be.appendValue(bv2)
				if len(be) > int(math.MaxUint8-1) {
					out.appendI32Op(OC_jz, int32(len(be)+1))
				} else {
					out.append(OC_jz8, OpCode(len(be)+1))
				}
				out.append(OC_pop)
				out.append(be...)
				bv = bvNone()
			} else {
				out.bland(&bv, bv2)
			}
		} else {
			break
		}
	}
	return bv, nil
}
func (c *Compiler) expBoolXor(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	return c.expOneOp(out, in, c.expBoolAnd, "^^", out.blxor, OC_blxor)
}
func (c *Compiler) expBoolOr(out *BytecodeExp, in *string) (BytecodeValue,
	error) {
	defer func(omp string) { c.maeOp = omp }(c.maeOp)
	if c.block != nil {
		return c.expOneOp(out, in, c.expBoolXor, "||", out.blor, OC_blor)
	}
	bv, err := c.expBoolXor(out, in)
	if err != nil {
		return bvNone(), err
	}
	for {
		if err := c.operator(in); err != nil {
			return bvNone(), err
		}
		if c.token == "||" {
			c.token = c.tokenizer(in)
			var be BytecodeExp
			bv2, err := c.expBoolXor(&be, in)
			if err != nil {
				return bvNone(), err
			}
			if bv.IsNone() || bv2.IsNone() {
				out.appendValue(bv)
				be.appendValue(bv2)
				if len(be) > int(math.MaxUint8-1) {
					out.appendI32Op(OC_jnz, int32(len(be)+1))
				} else {
					out.append(OC_jnz8, OpCode(len(be)+1))
				}
				out.append(OC_pop)
				out.append(be...)
				bv = bvNone()
			} else {
				out.bland(&bv, bv2)
			}
		} else {
			break
		}
	}
	return bv, nil
}
func (c *Compiler) typedExp(ef expFunc, in *string,
	vt ValueType) (BytecodeExp, error) {
	c.token = c.tokenizer(in)
	var be BytecodeExp
	bv, err := ef(&be, in)
	if err != nil {
		return nil, err
	}
	if !bv.IsNone() {
		switch vt {
		case VT_Float:
			bv.SetF(bv.ToF())
		case VT_Int:
			bv.SetI(bv.ToI())
		case VT_Bool:
			bv.SetB(bv.ToB())
		}
		be.appendValue(bv)
	}
	return be, nil
}
func (c *Compiler) argExpression(in *string, vt ValueType) (BytecodeExp,
	error) {
	be, err := c.typedExp(c.expBoolOr, in, vt)
	if err != nil {
		return nil, err
	}
	if len(c.token) > 0 && c.token != "," {
		return nil, Error(c.token + "が不正です")
	}
	return be, nil
}
func (c *Compiler) fullExpression(in *string, vt ValueType) (BytecodeExp,
	error) {
	be, err := c.typedExp(c.expBoolOr, in, vt)
	if err != nil {
		return nil, err
	}
	if len(c.token) > 0 {
		return nil, Error(c.token + "が不正です")
	}
	return be, nil
}
func (c *Compiler) parseSection(
	sctrl func(name, data string) error) (IniSection, bool, error) {
	is := NewIniSection()
	_type, persistent, ignorehitpause := true, true, true
	for ; c.i < len(c.lines); (c.i)++ {
		line := strings.ToLower(strings.TrimSpace(
			strings.SplitN(c.lines[c.i], ";", 2)[0]))
		if len(line) > 0 && line[0] == '[' {
			c.i--
			break
		}
		var name, data string
		if len(line) >= 3 && strings.ToLower(line[:3]) == "var" {
			name, data = "var", line
		} else if len(line) >= 4 && strings.ToLower(line[:4]) == "fvar" {
			name, data = "fvar", line
		} else if len(line) >= 6 && strings.ToLower(line[:6]) == "sysvar" {
			name, data = "sysvar", line
		} else if len(line) >= 7 && strings.ToLower(line[:7]) == "sysfvar" {
			name, data = "sysfvar", line
		} else {
			ia := strings.IndexAny(line, "= \t")
			if ia > 0 {
				name = strings.ToLower(line[:ia])
				ia = strings.Index(line, "=")
				if ia >= 0 {
					data = strings.TrimSpace(line[ia+1:])
				}
			}
		}
		if len(name) > 0 {
			_, ok := is[name]
			if ok && (len(name) < 7 || name[:7] != "trigger") {
				if sys.ignoreMostErrors {
					continue
				}
				return nil, false, Error(name + "が重複しています")
			}
			if sctrl != nil {
				switch name {
				case "type":
					if !_type {
						continue
					}
					_type = false
				case "persistent":
					if !persistent {
						continue
					}
					persistent = false
				case "ignorehitpause":
					if !ignorehitpause {
						continue
					}
					ignorehitpause = false
				default:
					if len(name) < 7 || name[:7] != "trigger" {
						is[name] = data
						continue
					}
				}
				if err := sctrl(name, data); err != nil {
					return nil, false, err
				}
			} else {
				is[name] = data
			}
		}
	}
	return is, !ignorehitpause, nil
}
func (c *Compiler) stateSec(is IniSection, f func() error) error {
	if err := f(); err != nil {
		return err
	}
	if !sys.ignoreMostErrors {
		var str string
		for k, _ := range is {
			if len(str) > 0 {
				str += ", "
			}
			str += k
		}
		if len(str) > 0 {
			return Error(str + "は無効なキー名です")
		}
	}
	return nil
}
func (c *Compiler) stateParam(is IniSection, name string,
	f func(string) error) error {
	data, ok := is[name]
	if ok {
		if err := f(data); err != nil {
			return Error(data + "\n" + name + ": " + err.Error())
		}
		delete(is, name)
	}
	return nil
}
func (c *Compiler) exprs(data string, vt ValueType,
	numArg int) ([]BytecodeExp, error) {
	bes := []BytecodeExp{}
	for n := 1; n <= numArg; n++ {
		var be BytecodeExp
		var err error
		if n < numArg {
			be, err = c.argExpression(&data, vt)
		} else {
			be, err = c.fullExpression(&data, vt)
		}
		if err != nil {
			return nil, err
		}
		bes = append(bes, be)
		if c.token != "," {
			break
		}
	}
	return bes, nil
}
func (c *Compiler) scAdd(sc *StateControllerBase, id byte,
	data string, vt ValueType, numArg int, topbe ...BytecodeExp) error {
	bes, err := c.exprs(data, vt, numArg)
	if err != nil {
		return err
	}
	sc.add(id, append(topbe, bes...))
	return nil
}
func (c *Compiler) paramValue(is IniSection, sc *StateControllerBase,
	paramname string, id byte, vt ValueType, numArg int, mandatory bool) error {
	f := false
	if err := c.stateParam(is, paramname, func(data string) error {
		f = true
		return c.scAdd(sc, id, data, vt, numArg)
	}); err != nil {
		return err
	}
	if mandatory && !f {
		return Error(paramname + "が指定されていません")
	}
	return nil
}
func (c *Compiler) paramPostye(is IniSection, sc *StateControllerBase,
	id byte) error {
	return c.stateParam(is, "postype", func(data string) error {
		if len(data) == 0 {
			return Error("値が指定されていません")
		}
		var pt PosType
		if len(data) >= 2 && strings.ToLower(data[:2]) == "p2" {
			pt = PT_P2
		} else {
			switch strings.ToLower(data)[0] {
			case 'p':
				pt = PT_P1
			case 'f':
				pt = PT_F
			case 'b':
				pt = PT_B
			case 'l':
				pt = PT_L
			case 'r':
				pt = PT_R
			case 'n':
				pt = PT_N
			default:
				return Error(data + "が無効な値です")
			}
		}
		sc.add(id, sc.iToExp(int32(pt)))
		return nil
	})
}
func (c *Compiler) paramTrans(is IniSection, sc *StateControllerBase,
	prefix string, id byte, afterImage bool) error {
	return c.stateParam(is, prefix+"trans", func(data string) error {
		if len(data) == 0 {
			return Error("値が指定されていません")
		}
		tt := TT_default
		data = strings.ToLower(data)
		switch data {
		case "none":
			tt = TT_none
		case "add1":
			tt = TT_add1
		case "sub":
			tt = TT_sub
		default:
			_error := false
			if afterImage {
				if len(data) >= 3 && data[:3] == "add" {
					tt = TT_add
				} else {
					_error = true
				}
			} else {
				switch data {
				case "default":
					tt = TT_default
				case "add":
					tt = TT_add
				case "addalpha", "alpha":
					tt = TT_alpha
				default:
					_error = true
				}
			}
			if _error && (!afterImage || !sys.ignoreMostErrors) {
				return Error(data + "が無効な値です")
			}
		}
		var exp []BytecodeExp
		b := false
		if !afterImage {
			if err := c.stateParam(is, prefix+"alpha", func(data string) error {
				b = true
				bes, err := c.exprs(data, VT_Int, 2)
				if err != nil {
					return err
				}
				if tt == TT_add1 {
					exp = make([]BytecodeExp, 4) // 長さ4にする
				} else if tt == TT_add || tt == TT_alpha {
					exp = make([]BytecodeExp, 3) // 長さ3にする
				} else {
					exp = make([]BytecodeExp, 2)
				}
				exp[0] = bes[0]
				if len(exp) == 2 {
					exp[0].append(OC_pop)
					switch tt {
					case TT_none:
						exp[0].appendValue(BytecodeInt(255))
					case TT_sub:
						exp[0].appendValue(BytecodeInt(1))
					default:
						exp[0].appendValue(BytecodeInt(-1))
					}
				}
				if len(bes) > 1 {
					exp[1] = bes[1]
					if tt != TT_alpha && tt != TT_add1 {
						exp[1].append(OC_pop)
					}
				}
				switch tt {
				case TT_alpha, TT_add1:
					if len(bes) <= 1 {
						exp[1].appendValue(BytecodeInt(255))
					}
				case TT_add, TT_sub:
					exp[1].appendValue(BytecodeInt(255))
				default:
					exp[1].appendValue(BytecodeInt(0))
				}
				return nil
			}); err != nil {
				return err
			}
		}
		if !b {
			switch tt {
			case TT_none:
				exp = sc.iToExp(255, 0)
			case TT_add:
				exp = sc.iToExp(255, 255)
			case TT_add1:
				exp = sc.iToExp(255, ^255)
			case TT_sub:
				exp = sc.iToExp(1, 255)
			default:
				exp = sc.iToExp(-1, 0)
			}
		}
		sc.add(id, exp)
		return nil
	})
}
func (c *Compiler) stateDef(is IniSection, sbc *StateBytecode) error {
	return c.stateSec(is, func() error {
		sc := newStateControllerBase()
		if err := c.stateParam(is, "type", func(data string) error {
			if len(data) == 0 {
				return Error("値が指定されていません")
			}
			switch strings.ToLower(data)[0] {
			case 's':
				sbc.stateType = ST_S
			case 'c':
				sbc.stateType = ST_C
			case 'a':
				sbc.stateType = ST_A
			case 'l':
				sbc.stateType = ST_L
			case 'u':
				sbc.stateType = ST_U
			default:
				return Error(data + "が無効な値です")
			}
			return nil
		}); err != nil {
			return err
		}
		if err := c.stateParam(is, "movetype", func(data string) error {
			if len(data) == 0 {
				return Error("値が指定されていません")
			}
			switch strings.ToLower(data)[0] {
			case 'i':
				sbc.moveType = MT_I
			case 'a':
				sbc.moveType = MT_A
			case 'h':
				sbc.moveType = MT_H
			case 'u':
				sbc.moveType = MT_U
			default:
				return Error(data + "が無効な値です")
			}
			return nil
		}); err != nil {
			return err
		}
		if err := c.stateParam(is, "physics", func(data string) error {
			if len(data) == 0 {
				return Error("値が指定されていません")
			}
			switch strings.ToLower(data)[0] {
			case 's':
				sbc.physics = ST_S
			case 'c':
				sbc.physics = ST_C
			case 'a':
				sbc.physics = ST_A
			case 'n':
				sbc.physics = ST_N
			case 'u':
				sbc.physics = ST_U
			default:
				return Error(data + "が無効な値です")
			}
			return nil
		}); err != nil {
			return err
		}
		b := false
		if err := c.stateParam(is, "hitcountpersist", func(data string) error {
			b = true
			return c.scAdd(sc, stateDef_hitcountpersist, data, VT_Bool, 1)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(stateDef_hitcountpersist, sc.iToExp(0))
		}
		b = false
		if err := c.stateParam(is, "movehitpersist", func(data string) error {
			b = true
			return c.scAdd(sc, stateDef_movehitpersist, data, VT_Bool, 1)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(stateDef_movehitpersist, sc.iToExp(0))
		}
		b = false
		if err := c.stateParam(is, "hitdefpersist", func(data string) error {
			b = true
			return c.scAdd(sc, stateDef_hitdefpersist, data, VT_Bool, 1)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(stateDef_hitdefpersist, sc.iToExp(0))
		}
		if err := c.paramValue(is, sc, "sprpriority",
			stateDef_sprpriority, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "facep2",
			stateDef_facep2, VT_Bool, 1, false); err != nil {
			return err
		}
		b = false
		if err := c.stateParam(is, "juggle", func(data string) error {
			b = true
			return c.scAdd(sc, stateDef_juggle, data, VT_Int, 1)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(stateDef_juggle, sc.iToExp(0))
		}
		if err := c.paramValue(is, sc, "velset",
			stateDef_velset, VT_Float, 3, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "anim",
			stateDef_anim, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "ctrl",
			stateDef_ctrl, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "poweradd",
			stateDef_poweradd, VT_Int, 1, false); err != nil {
			return err
		}
		sbc.stateDef = stateDef(*sc)
		return nil
	})
}
func (c *Compiler) hitBySub(is IniSection, sc *StateControllerBase) error {
	attr, two := int32(-1), false
	var err error
	if err = c.stateParam(is, "value", func(data string) error {
		attr, err = c.attr(data, false)
		return err
	}); err != nil {
		return err
	}
	if attr == -1 {
		if err = c.stateParam(is, "value2", func(data string) error {
			two = true
			attr, err = c.attr(data, false)
			return err
		}); err != nil {
			return err
		}
	}
	if attr == -1 {
		return Error("valueが指定されていません")
	}
	if err := c.paramValue(is, sc, "time",
		hitBy_time, VT_Int, 1, false); err != nil {
		return err
	}
	if two {
		sc.add(hitBy_value2, sc.iToExp(attr))
	} else {
		sc.add(hitBy_value, sc.iToExp(attr))
	}
	return nil
}
func (c *Compiler) hitBy(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*hitBy)(sc), c.stateSec(is, func() error {
		return c.hitBySub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) notHitBy(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*notHitBy)(sc), c.stateSec(is, func() error {
		return c.hitBySub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) assertSpecial(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*assertSpecial)(sc), c.stateSec(is, func() error {
		foo := func(data string) error {
			switch data {
			case "nostandguard":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_nostandguard)))
			case "nocrouchguard":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_nocrouchguard)))
			case "noairguard":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_noairguard)))
			case "noshadow":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_noshadow)))
			case "invisible":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_invisible)))
			case "unguardable":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_unguardable)))
			case "nojugglecheck":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_nojugglecheck)))
			case "noautoturn":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_noautoturn)))
			case "nowalk":
				sc.add(assertSpecial_flag, sc.iToExp(int32(CSF_nowalk)))
			case "intro":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_intro)))
			case "roundnotover":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_roundnotover)))
			case "nomusic":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_nomusic)))
			case "nobardisplay":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_nobardisplay)))
			case "nobg":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_nobg)))
			case "nofg":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_nofg)))
			case "globalnoshadow":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_globalnoshadow)))
			case "timerfreeze":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_timerfreeze)))
			case "nokosnd":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_nokosnd)))
			case "nokoslow":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_nokoslow)))
			case "noko":
				sc.add(assertSpecial_flag_g, sc.iToExp(int32(GSF_noko)))
			default:
				return Error(data + "が無効な値です")
			}
			return nil
		}
		f := false
		if err := c.stateParam(is, "flag", func(data string) error {
			f = true
			return foo(data)
		}); err != nil {
			return err
		}
		if !f {
			return Error("flagが指定されていません")
		}
		if err := c.stateParam(is, "flag2", func(data string) error {
			return foo(data)
		}); err != nil {
			return err
		}
		if err := c.stateParam(is, "flag3", func(data string) error {
			return foo(data)
		}); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) playSnd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*playSnd)(sc), c.stateSec(is, func() error {
		f := false
		if err := c.stateParam(is, "value", func(data string) error {
			f = true
			fflg := false
			if len(data) > 0 {
				switch data[0] {
				case 'F', 'f':
					fflg = true
					data = data[1:]
				case 'S', 's':
					data = data[1:]
				}
			}
			return c.scAdd(sc, playSnd_value, data, VT_Int, 2,
				sc.iToExp(Btoi(fflg))...)
		}); err != nil {
			return err
		}
		if !f {
			return Error("valueが指定されていません")
		}
		if err := c.paramValue(is, sc, "channel",
			playSnd_channel, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "lowpriority",
			playSnd_lowpriority, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pan",
			playSnd_pan, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "abspan",
			playSnd_abspan, VT_Float, 1, false); err != nil {
			return err
		}
		var volname string
		if sys.cgi[c.playerNo].ver[0] == 1 {
			volname = "volumescale"
		} else {
			volname = "volume"
		}
		if err := c.paramValue(is, sc, volname,
			playSnd_volume, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "freqmul",
			playSnd_freqmul, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "loop",
			playSnd_loop, VT_Bool, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) changeStateSub(is IniSection,
	sc *StateControllerBase) error {
	if err := c.paramValue(is, sc, "value",
		changeState_value, VT_Int, 1, true); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "ctrl",
		changeState_ctrl, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "anim",
		changeState_anim, VT_Int, 1, false); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) changeState(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*changeState)(sc), c.stateSec(is, func() error {
		return c.changeStateSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) selfState(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*selfState)(sc), c.stateSec(is, func() error {
		return c.changeStateSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) tagIn(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*tagIn)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "stateno",
			tagIn_stateno, VT_Int, 1, true); err != nil {
			return err
		}
		f := false
		if err := c.stateParam(is, "partnerstateno", func(data string) error {
			f = true
			return c.scAdd(sc, tagIn_partnerstateno, data, VT_Int, 1)
		}); err != nil {
			return err
		}
		if !f {
			sc.add(tagIn_partnerstateno, sc.iToExp(-1))
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) tagOut(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*tagOut)(sc), c.stateSec(is, func() error {
		sc.add(tagOut_, nil)
		return nil
	})
	return *ret, err
}
func (c *Compiler) destroySelf(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*destroySelf)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "recursive",
			destroySelf_recursive, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "removeexplods",
			destroySelf_removeexplods, VT_Bool, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) changeAnimSub(is IniSection,
	sc *StateControllerBase) error {
	if err := c.paramValue(is, sc, "elem",
		changeAnim_elem, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "value",
		changeAnim_value, VT_Int, 1, true); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) changeAnim(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*changeAnim)(sc), c.stateSec(is, func() error {
		return c.changeAnimSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) changeAnim2(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*changeAnim2)(sc), c.stateSec(is, func() error {
		return c.changeAnimSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) helper(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*helper)(sc), c.stateSec(is, func() error {
		if err := c.stateParam(is, "helpertype", func(data string) error {
			if len(data) == 0 {
				return Error("値が指定されていません")
			}
			switch strings.ToLower(data)[0] {
			case 'n':
			case 'p':
				sc.add(helper_helpertype, sc.iToExp(1))
			default:
				return Error(data + "が無効な値です")
			}
			return nil
		}); err != nil {
			return err
		}
		if err := c.stateParam(is, "name", func(data string) error {
			if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
				return Error("\"で囲まれていません")
			}
			sc.add(helper_name, sc.beToExp(BytecodeExp(data[1:len(data)-1])))
			return nil
		}); err != nil {
			return err
		}
		if err := c.paramPostye(is, sc, helper_postype); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "ownpal",
			helper_ownpal, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.xscale",
			helper_size_xscale, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.yscale",
			helper_size_yscale, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.ground.back",
			helper_size_ground_back, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.ground.front",
			helper_size_ground_front, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.air.back",
			helper_size_air_back, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.air.front",
			helper_size_air_front, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.height",
			helper_size_height, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.proj.doscale",
			helper_size_proj_doscale, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.head.pos",
			helper_size_head_pos, VT_Int, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.mid.pos",
			helper_size_mid_pos, VT_Int, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "size.shadowoffset",
			helper_size_shadowoffset, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "stateno",
			helper_stateno, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "keyctrl",
			helper_keyctrl, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "id",
			helper_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pos",
			helper_pos, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "facing",
			helper_facing, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pausemovetime",
			helper_pausemovetime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "supermovetime",
			helper_supermovetime, VT_Int, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) ctrlSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*ctrlSet)(sc), c.stateSec(is, func() error {
		return c.paramValue(is, sc, "value", ctrlSet_value, VT_Bool, 1, true)
	})
	return *ret, err
}
func (c *Compiler) explodSub(is IniSection,
	sc *StateControllerBase) error {
	if err := c.paramValue(is, sc, "remappal",
		explod_remappal, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "id",
		explod_id, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramPostye(is, sc, explod_postype); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "facing",
		explod_facing, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "vfacing",
		explod_vfacing, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "pos",
		explod_pos, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "random",
		explod_random, VT_Float, 2, false); err != nil {
		return err
	}
	f := false
	if err := c.stateParam(is, "vel", func(data string) error {
		f = true
		return c.scAdd(sc, explod_velocity, data, VT_Float, 2)
	}); err != nil {
		return err
	}
	if !f {
		if err := c.paramValue(is, sc, "velocity",
			explod_velocity, VT_Float, 2, false); err != nil {
			return err
		}
	}
	if err := c.paramValue(is, sc, "accel",
		explod_accel, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "scale",
		explod_scale, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "bindtime",
		explod_bindtime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "removetime",
		explod_removetime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "supermove",
		explod_supermove, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "supermovetime",
		explod_supermovetime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "pausemovetime",
		explod_pausemovetime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "sprpriority",
		explod_sprpriority, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "ontop",
		explod_ontop, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "shadow",
		explod_shadow, VT_Int, 3, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "removeongethit",
		explod_removeongethit, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramTrans(is, sc, "", explod_trans, false); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) explod(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, ihp bool) (StateController, error) {
	ret, err := (*explod)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "ownpal",
			explod_ownpal, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.explodSub(is, sc); err != nil {
			return err
		}
		if err := c.stateParam(is, "anim", func(data string) error {
			fflg := false
			if len(data) > 0 && strings.ToLower(data)[0] == 'f' {
				fflg = true
				data = data[1:]
			}
			return c.scAdd(sc, explod_anim, data, VT_Int, 1,
				sc.iToExp(Btoi(fflg))...)
		}); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "angle",
			explod_angle, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "yangle",
			explod_yangle, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "xangle",
			explod_xangle, VT_Float, 1, false); err != nil {
			return err
		}
		if ihp && !c.block.ignorehitpause {
			sc.add(explod_ignorehitpause, sc.iToExp(0))
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) modifyExplod(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*modifyExplod)(sc), c.stateSec(is, func() error {
		if err := c.explodSub(is, sc); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "angle",
			explod_angle, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "yangle",
			explod_yangle, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "xangle",
			explod_xangle, VT_Float, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) gameMakeAnim(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*gameMakeAnim)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "pos",
			gameMakeAnim_pos, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "random",
			gameMakeAnim_random, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "under",
			gameMakeAnim_under, VT_Bool, 1, false); err != nil {
			return err
		}
		b := false
		anim := func(data string) error {
			b = true
			fflg := true
			if len(data) > 0 && strings.ToLower(data)[0] == 's' {
				fflg = false
				data = data[1:]
			}
			return c.scAdd(sc, gameMakeAnim_anim, data, VT_Int, 1,
				sc.iToExp(Btoi(fflg))...)
		}
		if err := c.stateParam(is, "anim", func(data string) error {
			return anim(data)
		}); err != nil {
			return err
		}
		if !b {
			if err := c.stateParam(is, "value", func(data string) error {
				return anim(data)
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) posSetSub(is IniSection,
	sc *StateControllerBase) error {
	if err := c.paramValue(is, sc, "x",
		posSet_x, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "y",
		posSet_y, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "z",
		posSet_z, VT_Float, 1, false); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) posSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*posSet)(sc), c.stateSec(is, func() error {
		return c.posSetSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) posAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*posAdd)(sc), c.stateSec(is, func() error {
		return c.posSetSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) velSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*velSet)(sc), c.stateSec(is, func() error {
		return c.posSetSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) velAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*velAdd)(sc), c.stateSec(is, func() error {
		return c.posSetSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) velMul(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*velMul)(sc), c.stateSec(is, func() error {
		return c.posSetSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) palFXSub(is IniSection,
	sc *StateControllerBase, prefix string) error {
	if err := c.paramValue(is, sc, prefix+"time",
		palFX_time, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"color",
		palFX_color, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.stateParam(is, prefix+"add", func(data string) error {
		bes, err := c.exprs(data, VT_Int, 3)
		if err != nil {
			return err
		}
		if len(bes) < 3 {
			return Error(prefix + "addの要素が足りません")
		}
		sc.add(palFX_add, bes)
		return nil
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, prefix+"mul", func(data string) error {
		bes, err := c.exprs(data, VT_Int, 3)
		if err != nil {
			return err
		}
		if len(bes) < 3 {
			return Error(prefix + "mulの要素が足りません")
		}
		sc.add(palFX_mul, bes)
		return nil
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, prefix+"sinadd", func(data string) error {
		bes, err := c.exprs(data, VT_Int, 4)
		if err != nil {
			return err
		}
		if len(bes) < 3 {
			return Error(prefix + "sinaddの要素が足りません")
		}
		sc.add(palFX_sinadd, bes)
		return nil
	}); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"invertall",
		palFX_invertall, VT_Bool, 1, false); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) palFX(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*palFX)(sc), c.stateSec(is, func() error {
		return c.palFXSub(is, sc, "")
	})
	return *ret, err
}
func (c *Compiler) allPalFX(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*allPalFX)(sc), c.stateSec(is, func() error {
		return c.palFXSub(is, sc, "")
	})
	return *ret, err
}
func (c *Compiler) bgPalFX(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*bgPalFX)(sc), c.stateSec(is, func() error {
		return c.palFXSub(is, sc, "")
	})
	return *ret, err
}
func (c *Compiler) afterImageSub(is IniSection,
	sc *StateControllerBase, prefix string) error {
	if err := c.paramTrans(is, sc, prefix,
		afterImage_trans, true); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"time",
		afterImage_time, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"length",
		afterImage_length, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"timegap",
		afterImage_timegap, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"framegap",
		afterImage_framegap, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"palcolor",
		afterImage_palcolor, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"palinvertall",
		afterImage_palinvertall, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"palbright",
		afterImage_palbright, VT_Int, 3, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"palcontrast",
		afterImage_palcontrast, VT_Int, 3, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"palpostbright",
		afterImage_palpostbright, VT_Int, 3, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"paladd",
		afterImage_paladd, VT_Int, 3, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, prefix+"palmul",
		afterImage_palmul, VT_Float, 3, false); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) afterImage(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*afterImage)(sc), c.stateSec(is, func() error {
		return c.afterImageSub(is, sc, "")
	})
	return *ret, err
}
func (c *Compiler) afterImageTime(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*afterImageTime)(sc), c.stateSec(is, func() error {
		b := false
		if err := c.stateParam(is, "time", func(data string) error {
			b = true
			return c.scAdd(sc, afterImageTime_time, data, VT_Int, 1)
		}); err != nil {
			return err
		}
		if !b {
			if err := c.stateParam(is, "value", func(data string) error {
				b = true
				return c.scAdd(sc, afterImageTime_time, data, VT_Int, 1)
			}); err != nil {
				return err
			}
			if !b {
				sc.add(afterImageTime_time, sc.iToExp(0))
			}
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) hitDefSub(is IniSection,
	sc *StateControllerBase) error {
	if err := c.stateParam(is, "attr", func(data string) error {
		attr, err := c.attr(data, true)
		if err != nil {
			return err
		}
		sc.add(hitDef_attr, sc.iToExp(attr))
		return nil
	}); err != nil {
		return err
	}
	hflg := func(id byte, data string) error {
		var flg int32
		for _, c := range data {
			switch c {
			case 'H', 'h':
				flg |= int32(ST_S)
			case 'L', 'l':
				flg |= int32(ST_C)
			case 'M', 'm':
				flg |= int32(ST_S | ST_C)
			case 'A', 'a':
				flg |= int32(ST_A)
			case 'F', 'f':
				flg |= int32(ST_F)
			case 'D', 'd':
				flg |= int32(ST_D)
			case 'P', 'p':
				flg |= int32(ST_P)
			case '-':
				flg |= int32(MT_MNS)
			case '+':
				flg |= int32(MT_PLS)
			}
		}
		sc.add(id, sc.iToExp(flg))
		return nil
	}
	if err := c.stateParam(is, "guardflag", func(data string) error {
		return hflg(hitDef_guardflag, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "hitflag", func(data string) error {
		return hflg(hitDef_hitflag, data)
	}); err != nil {
		return err
	}
	htyp := func(id byte, data string) error {
		if len(data) == 0 {
			return Error("値が指定されていません")
		}
		var ht HitType
		switch data[0] {
		case 'H', 'h':
			ht = HT_High
		case 'L', 'l':
			ht = HT_Low
		case 'T', 't':
			ht = HT_Trip
		case 'N', 'n':
			ht = HT_None
		default:
			return Error(data + "が無効な値です")
		}
		sc.add(id, sc.iToExp(int32(ht)))
		return nil
	}
	if err := c.stateParam(is, "ground.type", func(data string) error {
		return htyp(hitDef_ground_type, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "air.type", func(data string) error {
		return htyp(hitDef_air_type, data)
	}); err != nil {
		return err
	}
	reac := func(id byte, data string) error {
		if len(data) == 0 {
			return Error("値が指定されていません")
		}
		var ra Reaction
		switch data[0] {
		case 'L', 'l':
			ra = RA_Light
		case 'M', 'm':
			ra = RA_Medium
		case 'H', 'h':
			ra = RA_Hard
		case 'B', 'b':
			ra = RA_Back
		case 'U', 'u':
			ra = RA_Up
		case 'D', 'd':
			ra = RA_Diagup
		default:
			return Error(data + "が無効な値です")
		}
		sc.add(id, sc.iToExp(int32(ra)))
		return nil
	}
	if err := c.stateParam(is, "animtype", func(data string) error {
		return reac(hitDef_animtype, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "air.animtype", func(data string) error {
		return reac(hitDef_air_animtype, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "fall.animtype", func(data string) error {
		return reac(hitDef_fall_animtype, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "affectteam", func(data string) error {
		if len(data) == 0 {
			return Error("値が指定されていません")
		}
		var at int32
		switch data[0] {
		case 'E', 'e':
			at = 1
		case 'B', 'b':
			at = 0
		case 'F', 'f':
			at = -1
		default:
			return Error(data + "が無効な値です")
		}
		sc.add(hitDef_affectteam, sc.iToExp(at))
		return nil
	}); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "id",
		hitDef_id, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "chainid",
		hitDef_chainid, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "nochainid",
		hitDef_nochainid, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.kill",
		hitDef_guard_kill, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.kill",
		hitDef_fall_kill, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "hitonce",
		hitDef_hitonce, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "air.juggle",
		hitDef_air_juggle, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "getpower",
		hitDef_getpower, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "damage",
		hitDef_damage, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "givepower",
		hitDef_givepower, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "numhits",
		hitDef_numhits, VT_Int, 1, false); err != nil {
		return err
	}
	hsnd := func(id byte, data string) error {
		fflg := true
		if len(data) > 0 {
			switch data[0] {
			case 'F', 'f':
				data = data[1:]
			case 'S', 's':
				fflg = false
				data = data[1:]
			}
		}
		return c.scAdd(sc, id, data, VT_Int, 2, sc.iToExp(Btoi(fflg))...)
	}
	if err := c.stateParam(is, "hitsound", func(data string) error {
		return hsnd(hitDef_hitsound, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "guardsound", func(data string) error {
		return hsnd(hitDef_guardsound, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "priority", func(data string) error {
		be, err := c.argExpression(&data, VT_Int)
		if err != nil {
			return err
		}
		at := AT_Hit
		data = strings.TrimSpace(data)
		if c.token == "," && len(data) > 0 {
			switch data[0] {
			case 'H', 'h':
				at = AT_Hit
			case 'M', 'm':
				at = AT_Miss
			case 'D', 'd':
				at = AT_Dodge
			default:
				return Error(data + "が無効な値です")
			}
		}
		sc.add(hitDef_priority, append(sc.beToExp(be), sc.iToExp(int32(at))...))
		return nil
	}); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "p1stateno",
		hitDef_p1stateno, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "p2stateno",
		hitDef_p2stateno, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "p2getp1state",
		hitDef_p2getp1state, VT_Bool, 1, false); err != nil {
		return err
	}
	b := false
	if err := c.stateParam(is, "p1sprpriority", func(data string) error {
		b = true
		return c.scAdd(sc, hitDef_p1sprpriority, data, VT_Int, 1)
	}); err != nil {
		return err
	}
	if !b {
		if err := c.paramValue(is, sc, "sprpriority",
			hitDef_p1sprpriority, VT_Int, 1, false); err != nil {
			return err
		}
	}
	if err := c.paramValue(is, sc, "p2sprpriority",
		hitDef_p2sprpriority, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "forcestand",
		hitDef_forcestand, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "forcenofall",
		hitDef_forcenofall, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.damage",
		hitDef_fall_damage, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.xvelocity",
		hitDef_fall_xvelocity, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.yvelocity",
		hitDef_fall_yvelocity, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.recover",
		hitDef_fall_recover, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.recovertime",
		hitDef_fall_recovertime, VT_Int, 1, false); err != nil {
		return err
	}
	sprk := func(id byte, data string) error {
		fflg := true
		if len(data) > 0 && strings.ToLower(data)[0] == 's' {
			fflg = false
			data = data[1:]
		}
		return c.scAdd(sc, id, data, VT_Int, 1, sc.iToExp(Btoi(fflg))...)
	}
	if err := c.stateParam(is, "sparkno", func(data string) error {
		return sprk(hitDef_sparkno, data)
	}); err != nil {
		return err
	}
	if err := c.stateParam(is, "guard.sparkno", func(data string) error {
		return sprk(hitDef_guard_sparkno, data)
	}); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "sparkxy",
		hitDef_sparkxy, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "down.hittime",
		hitDef_down_hittime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "p1facing",
		hitDef_p1facing, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "p1getp2facing",
		hitDef_p1getp2facing, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "mindist",
		hitDef_mindist, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "maxdist",
		hitDef_maxdist, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "snap",
		hitDef_snap, VT_Float, 4, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "p2facing",
		hitDef_p2facing, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "air.hittime",
		hitDef_air_hittime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall",
		hitDef_fall, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "air.fall",
		hitDef_air_fall, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "air.cornerpush.veloff",
		hitDef_air_cornerpush_veloff, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "down.bounce",
		hitDef_down_bounce, VT_Bool, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "down.velocity",
		hitDef_down_velocity, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "down.cornerpush.veloff",
		hitDef_down_cornerpush_veloff, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "ground.hittime",
		hitDef_ground_hittime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.hittime",
		hitDef_guard_hittime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.dist",
		hitDef_guard_dist, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "pausetime",
		hitDef_pausetime, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.pausetime",
		hitDef_guard_pausetime, VT_Int, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "air.velocity",
		hitDef_air_velocity, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "airguard.velocity",
		hitDef_airguard_velocity, VT_Float, 2, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "ground.slidetime",
		hitDef_ground_slidetime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.slidetime",
		hitDef_guard_slidetime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.ctrltime",
		hitDef_guard_ctrltime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "airguard.ctrltime",
		hitDef_airguard_ctrltime, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.stateParam(is, "ground.velocity", func(data string) error {
		in := data
		if c.token = c.tokenizer(&in); c.token == "n" {
			if c.token = c.tokenizer(&in); len(c.token) > 0 && c.token != "," {
				return Error(c.token + "がエラーです")
			}
		} else {
			in = data
			be, err := c.argExpression(&in, VT_Float)
			if err != nil {
				return err
			}
			sc.add(hitDef_ground_velocity_x, sc.beToExp(be))
		}
		if c.token == "," {
			oldin := in
			if c.token = c.tokenizer(&in); c.token == "n" {
				if c.token = c.tokenizer(&in); len(c.token) > 0 {
					return Error(c.token + "がエラーです")
				}
			} else {
				in = oldin
				be, err := c.fullExpression(&in, VT_Float)
				if err != nil {
					return err
				}
				sc.add(hitDef_ground_velocity_y, sc.beToExp(be))
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.velocity",
		hitDef_guard_velocity, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "ground.cornerpush.veloff",
		hitDef_ground_cornerpush_veloff, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "guard.cornerpush.veloff",
		hitDef_guard_cornerpush_veloff, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "airguard.cornerpush.veloff",
		hitDef_airguard_cornerpush_veloff, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "yaccel",
		hitDef_yaccel, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.palFXSub(is, sc, "palfx."); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "envshake.time",
		hitDef_envshake_time, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "envshake.ampl",
		hitDef_envshake_ampl, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "envshake.phase",
		hitDef_envshake_phase, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "envshake.freq",
		hitDef_envshake_freq, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.envshake.time",
		hitDef_fall_envshake_time, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.envshake.ampl",
		hitDef_fall_envshake_ampl, VT_Int, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.envshake.phase",
		hitDef_fall_envshake_phase, VT_Float, 1, false); err != nil {
		return err
	}
	if err := c.paramValue(is, sc, "fall.envshake.freq",
		hitDef_fall_envshake_freq, VT_Float, 1, false); err != nil {
		return err
	}
	return nil
}
func (c *Compiler) hitDef(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*hitDef)(sc), c.stateSec(is, func() error {
		return c.hitDefSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) reversalDef(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*reversalDef)(sc), c.stateSec(is, func() error {
		attr := int32(-1)
		var err error
		if err = c.stateParam(is, "reversal.attr", func(data string) error {
			attr, err = c.attr(data, false)
			return err
		}); err != nil {
			return err
		}
		if attr == -1 {
			return Error("reversal.attrが指定されていません")
		}
		sc.add(reversalDef_reversal_attr, sc.iToExp(attr))
		return c.hitDefSub(is, sc)
	})
	return *ret, err
}
func (c *Compiler) projectile(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*projectile)(sc), c.stateSec(is, func() error {
		if err := c.paramPostye(is, sc, projectile_postype); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projid",
			projectile_projid, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projremove",
			projectile_projremove, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projremovetime",
			projectile_projremovetime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projshadow",
			projectile_projshadow, VT_Int, 3, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projmisstime",
			projectile_projmisstime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projhits",
			projectile_projhits, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projpriority",
			projectile_projpriority, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projhitanim",
			projectile_projhitanim, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projremanim",
			projectile_projremanim, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projcancelanim",
			projectile_projcancelanim, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "velocity",
			projectile_velocity, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "velmul",
			projectile_velmul, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "remvelocity",
			projectile_remvelocity, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "accel",
			projectile_accel, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projscale",
			projectile_projscale, VT_Float, 2, false); err != nil {
			return err
		}

		// hitdef部分
		if err := c.hitDefSub(is, sc); err != nil {
			return err
		}

		if err := c.paramValue(is, sc, "offset",
			projectile_offset, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projsprpriority",
			projectile_projsprpriority, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projstagebound",
			projectile_projstagebound, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projedgebound",
			projectile_projedgebound, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projheightbound",
			projectile_projheightbound, VT_Int, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "projanim",
			projectile_projanim, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "supermovetime",
			projectile_supermovetime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pausemovetime",
			projectile_pausemovetime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "ownpal",
			projectile_ownpal, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "remappal",
			projectile_remappal, VT_Int, 2, false); err != nil {
			return err
		}
		if err := c.afterImageSub(is, sc, "afterimage."); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) width(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*width)(sc), c.stateSec(is, func() error {
		b := false
		if err := c.stateParam(is, "edge", func(data string) error {
			b = true
			if len(data) == 0 {
				return nil
			}
			return c.scAdd(sc, width_edge, data, VT_Float, 2)
		}); err != nil {
			return err
		}
		if err := c.stateParam(is, "player", func(data string) error {
			b = true
			if len(data) == 0 {
				return nil
			}
			return c.scAdd(sc, width_player, data, VT_Float, 2)
		}); err != nil {
			return err
		}
		if !b {
			if err := c.paramValue(is, sc, "value",
				width_value, VT_Float, 2, true); err != nil {
				return err
			}
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) sprPriority(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*sprPriority)(sc), c.stateSec(is, func() error {
		return c.paramValue(is, sc, "value",
			sprPriority_value, VT_Int, 1, false)
	})
	return *ret, err
}
func (c *Compiler) varSetSub(is IniSection,
	sc *StateControllerBase, rd OpCode, oc OpCode) error {
	b, v, fv := false, false, false
	var value string
	if err := c.stateParam(is, "value", func(data string) error {
		b = true
		value = data
		return nil
	}); err != nil {
		return err
	}
	if b {
		var ve BytecodeExp
		if err := c.stateParam(is, "v", func(data string) (err error) {
			v = true
			ve, err = c.fullExpression(&data, VT_Int)
			return
		}); err != nil {
			return err
		}
		if !v {
			if err := c.stateParam(is, "fv", func(data string) (err error) {
				fv = true
				ve, err = c.fullExpression(&data, VT_Int)
				return
			}); err != nil {
				return err
			}
		}
		if v || fv {
			if len(ve) == 2 && ve[0] == OC_int8 && int8(ve[1]) >= 0 &&
				(v && ve[1] < NumVar || fv && ve[1] < NumFvar) {
				if oc == OC_st_var {
					if v {
						oc = OC_st_var0 + ve[1]
					} else {
						oc = OC_st_fvar0 + ve[1]
					}
				} else {
					if v {
						oc = OC_st_var0add + ve[1]
					} else {
						oc = OC_st_fvar0add + ve[1]
					}
				}
				ve = nil
			} else if oc == OC_st_var {
				if v {
					oc = OC_st_var
				} else {
					oc = OC_st_fvar
				}
			} else {
				if v {
					oc = OC_st_varadd
				} else {
					oc = OC_st_fvaradd
				}
			}
			var vt ValueType
			if v {
				vt = VT_Int
			} else {
				vt = VT_Float
			}
			in := value
			be, err := c.fullExpression(&in, vt)
			if err != nil {
				return Error(value + "\n" + "value: " + err.Error())
			}
			ve.append(be...)
			if rd != OC_rdreset {
				var tmp BytecodeExp
				tmp.appendI32Op(OC_nordrun, int32(len(ve)))
				ve.append(OC_st_, oc)
				ve = append(tmp, ve...)
				tmp = nil
				tmp.appendI32Op(rd, int32(len(ve)))
				ve = append(tmp, ve...)
			} else {
				ve.append(OC_st_, oc)
			}
			sc.add(varSet_, sc.beToExp(ve))
		}
		return nil
	}
	sys := false
	set := func(data string) error {
		data = strings.TrimSpace(data)
		if data[0] != '(' {
			return Error("'('がありません")
		}
		var be BytecodeExp
		c.token = c.tokenizer(&data)
		bv, err := c.expValue(&be, &data, false)
		if err != nil {
			return err
		}
		_else := false
		if !bv.IsNone() {
			i := bv.ToI()
			if i >= 0 && (!sys && v && i < int32(NumVar) ||
				!sys && fv && i < int32(NumFvar) || sys && v && i < int32(NumSysVar) ||
				sys && fv && i < int32(NumSysFvar)) {
				if v {
					if oc == OC_st_var {
						oc = OC_st_var0 + OpCode(i)
					} else {
						oc = OC_st_var0add + OpCode(i)
					}
					if sys {
						oc += NumVar
					}
				} else {
					if oc == OC_st_var {
						oc = OC_st_fvar0 + OpCode(i)
					} else {
						oc = OC_st_fvar0add + OpCode(i)
					}
					if sys {
						oc += NumFvar
					}
				}
			} else {
				be.appendValue(bv)
				_else = true
			}
		} else {
			_else = true
		}
		if _else {
			if oc == OC_st_var {
				if sys {
					if v {
						oc = OC_st_sysvar
					} else {
						oc = OC_st_sysfvar
					}
				} else {
					if v {
						oc = OC_st_var
					} else {
						oc = OC_st_fvar
					}
				}
			} else {
				if sys {
					if v {
						oc = OC_st_sysvaradd
					} else {
						oc = OC_st_sysfvaradd
					}
				} else {
					if v {
						oc = OC_st_varadd
					} else {
						oc = OC_st_fvaradd
					}
				}
			}
		}
		if len(c.token) == 0 || c.token[len(c.token)-1] != '=' {
			idx := strings.Index(data, "=")
			if idx < 0 {
				return Error("'='がありません")
			}
			data = data[idx+1:]
		}
		var vt ValueType
		if v {
			vt = VT_Int
		} else {
			vt = VT_Float
		}
		ve := be
		be, err = c.fullExpression(&data, vt)
		if err != nil {
			return err
		}
		ve.append(be...)
		if rd != OC_rdreset {
			var tmp BytecodeExp
			tmp.appendI32Op(OC_nordrun, int32(len(ve)))
			ve.append(OC_st_, oc)
			ve = append(tmp, ve...)
			tmp = nil
			tmp.appendI32Op(rd, int32(len(ve)))
			ve = append(tmp, ve...)
		} else {
			ve.append(OC_st_, oc)
		}
		sc.add(varSet_, sc.beToExp(ve))
		return nil
	}
	if err := c.stateParam(is, "var", func(data string) error {
		if data[0] != 'v' {
			return Error(data[:3] + "の'v'が小文字でありません")
		}
		b = true
		v = true
		return set(data[3:])
	}); err != nil {
		return err
	}
	if b {
		return nil
	}
	if err := c.stateParam(is, "fvar", func(data string) error {
		if rd == OC_rdreset && data[0] != 'f' {
			return Error(data[:4] + "の'f'が小文字でありません")
		}
		b = true
		fv = true
		return set(data[4:])
	}); err != nil {
		return err
	}
	if b {
		return nil
	}
	if err := c.stateParam(is, "sysvar", func(data string) error {
		if data[3] != 'v' {
			return Error(data[:6] + "の'v'が小文字でありません")
		}
		b = true
		v = true
		sys = true
		return set(data[6:])
	}); err != nil {
		return err
	}
	if b {
		return nil
	}
	if err := c.stateParam(is, "sysfvar", func(data string) error {
		b = true
		fv = true
		sys = true
		return set(data[7:])
	}); err != nil {
		return err
	}
	if b {
		return nil
	}
	return Error("valueが指定されていません")
}
func (c *Compiler) varSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*varSet)(sc), c.stateSec(is, func() error {
		return c.varSetSub(is, sc, OC_rdreset, OC_st_var)
	})
	return *ret, err
}
func (c *Compiler) varAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*varSet)(sc), c.stateSec(is, func() error {
		return c.varSetSub(is, sc, OC_rdreset, OC_st_varadd)
	})
	return *ret, err
}
func (c *Compiler) parentVarSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*varSet)(sc), c.stateSec(is, func() error {
		return c.varSetSub(is, sc, OC_parent, OC_st_var)
	})
	return *ret, err
}
func (c *Compiler) parentVarAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*varSet)(sc), c.stateSec(is, func() error {
		return c.varSetSub(is, sc, OC_parent, OC_st_varadd)
	})
	return *ret, err
}
func (c *Compiler) turn(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*turn)(sc), c.stateSec(is, func() error {
		sc.add(turn_, nil)
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetFacing(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetFacing)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetFacing_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "value",
			targetFacing_value, VT_Int, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetBind(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetBind)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetBind_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "time",
			targetBind_time, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pos",
			targetBind_pos, VT_Float, 2, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) bindToTarget(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*bindToTarget)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			bindToTarget_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "time",
			bindToTarget_time, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.stateParam(is, "pos", func(data string) error {
			be, err := c.argExpression(&data, VT_Float)
			if err != nil {
				return err
			}
			exp := sc.beToExp(be)
			if c.token != "," {
				sc.add(bindToTarget_pos, exp)
				return nil
			}
			if be, err = c.argExpression(&data, VT_Float); err != nil {
				return err
			}
			exp, data = append(exp, be), strings.TrimSpace(data)
			if c.token != "," || len(data) == 0 {
				sc.add(bindToTarget_pos, exp)
				return nil
			}
			var hmf HMF
			switch data[0] {
			case 'H', 'h':
				hmf = HMF_H
			case 'M', 'm':
				hmf = HMF_M
			case 'F', 'f':
				hmf = HMF_F
			default:
				return Error(data + "が無効な値です")
			}
			sc.add(bindToTarget_pos, append(exp, sc.iToExp(int32(hmf))...))
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetLifeAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetLifeAdd)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetLifeAdd_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "absolute",
			targetLifeAdd_absolute, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "kill",
			targetLifeAdd_kill, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "value",
			targetLifeAdd_value, VT_Int, 1, true); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetState(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetState)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetState_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "value",
			targetState_value, VT_Int, 1, true); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetVelSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetVelSet)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetVelSet_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "x",
			targetVelSet_x, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "y",
			targetVelSet_y, VT_Float, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetVelAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetVelAdd)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetVelAdd_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "x",
			targetVelAdd_x, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "y",
			targetVelAdd_y, VT_Float, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetPowerAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetPowerAdd)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "id",
			targetPowerAdd_id, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "value",
			targetPowerAdd_value, VT_Int, 1, true); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) targetDrop(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*targetDrop)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "excludeid",
			targetDrop_excludeid, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "keepone",
			targetDrop_keepone, VT_Bool, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) lifeAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*lifeAdd)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "absolute",
			lifeAdd_absolute, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "kill",
			lifeAdd_kill, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "value",
			lifeAdd_value, VT_Int, 1, true); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) lifeSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*lifeSet)(sc), c.stateSec(is, func() error {
		return c.paramValue(is, sc, "value", lifeSet_value, VT_Int, 1, true)
	})
	return *ret, err
}
func (c *Compiler) powerAdd(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*powerAdd)(sc), c.stateSec(is, func() error {
		return c.paramValue(is, sc, "value", powerAdd_value, VT_Int, 1, true)
	})
	return *ret, err
}
func (c *Compiler) powerSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*powerSet)(sc), c.stateSec(is, func() error {
		return c.paramValue(is, sc, "value", powerSet_value, VT_Int, 1, true)
	})
	return *ret, err
}
func (c *Compiler) hitVelSet(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*hitVelSet)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "x",
			hitVelSet_x, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "y",
			hitVelSet_y, VT_Bool, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) screenBound(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*screenBound)(sc), c.stateSec(is, func() error {
		b := false
		if err := c.stateParam(is, "value", func(data string) error {
			b = true
			return c.scAdd(sc, screenBound_value, data, VT_Bool, 1)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(screenBound_value, sc.iToExp(0))
		}
		b = false
		if err := c.stateParam(is, "movecamera", func(data string) error {
			b = true
			return c.scAdd(sc, screenBound_movecamera, data, VT_Bool, 2)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(screenBound_value, append(sc.iToExp(0), sc.iToExp(0)...))
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) posFreeze(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*posFreeze)(sc), c.stateSec(is, func() error {
		b := false
		if err := c.stateParam(is, "value", func(data string) error {
			b = true
			return c.scAdd(sc, posFreeze_value, data, VT_Bool, 1)
		}); err != nil {
			return err
		}
		if !b {
			sc.add(posFreeze_value, sc.iToExp(1))
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) envShake(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*envShake)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "time",
			envShake_time, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "ampl",
			envShake_ampl, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "phase",
			envShake_phase, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "freq",
			envShake_freq, VT_Float, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) hitOverride(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*hitOverride)(sc), c.stateSec(is, func() error {
		if err := c.stateParam(is, "attr", func(data string) error {
			attr, err := c.attr(data, false)
			if err != nil {
				return err
			}
			sc.add(hitOverride_attr, sc.iToExp(attr))
			return nil
		}); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "slot",
			hitOverride_slot, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "stateno",
			hitOverride_stateno, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "time",
			hitOverride_time, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "forceair",
			hitOverride_forceair, VT_Bool, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) pause(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*pause)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "time",
			pause_time, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "movetime",
			pause_movetime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pausebg",
			pause_pausebg, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "endcmdbuftime",
			pause_endcmdbuftime, VT_Int, 1, false); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) superPause(is IniSection, sbc *StateBytecode,
	sc *StateControllerBase, _ bool) (StateController, error) {
	ret, err := (*superPause)(sc), c.stateSec(is, func() error {
		if err := c.paramValue(is, sc, "time",
			superPause_time, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "movetime",
			superPause_movetime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pausebg",
			superPause_pausebg, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "endcmdbuftime",
			superPause_endcmdbuftime, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "darken",
			superPause_darken, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.stateParam(is, "anim", func(data string) error {
			fflg := true
			if len(data) > 0 {
				data = strings.ToLower(data)
				if data[0] == 's' {
					fflg = false
					data = data[1:]
				} else if data[0] == 'f' {
					data = data[1:]
				}
			}
			return c.scAdd(sc, superPause_anim, data, VT_Int, 1,
				sc.iToExp(Btoi(fflg))...)
		}); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "pos",
			superPause_pos, VT_Float, 2, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "p2defmul",
			superPause_p2defmul, VT_Float, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "poweradd",
			superPause_poweradd, VT_Int, 1, false); err != nil {
			return err
		}
		if err := c.paramValue(is, sc, "unhittable",
			superPause_unhittable, VT_Bool, 1, false); err != nil {
			return err
		}
		if err := c.stateParam(is, "sound", func(data string) error {
			fflg := true
			if len(data) > 0 {
				data = strings.ToLower(data)
				if data[0] == 's' {
					fflg = false
					data = data[1:]
				} else if data[0] == 'f' {
					data = data[1:]
				}
			}
			return c.scAdd(sc, superPause_sound, data, VT_Int, 2,
				sc.iToExp(Btoi(fflg))...)
		}); err != nil {
			return err
		}
		return nil
	})
	return *ret, err
}
func (c *Compiler) stateCompile(bc *Bytecode, filename, def string) error {
	var str string
	fnz := filename
	if err := LoadFile(&filename, def, func(filename string) error {
		var err error
		str, err = LoadText(filename)
		return err
	}); err != nil {
		fnz += ".zss"
		if err := LoadFile(&fnz, def, func(filename string) error {
			b, err := ioutil.ReadFile(filename)
			if err != nil {
				return err
			}
			str = string(b)
			return nil
		}); err == nil {
			return c.stateCompileZ(bc, fnz, str)
		}
		return err
	}
	c.lines, c.i = SplitAndTrim(str, "\n"), 0
	errmes := func(err error) error {
		return Error(fmt.Sprintf("%v:%v:\n%v", filename, c.i+1, err.Error()))
	}
	existInThisFile := make(map[int32]bool)
	for ; c.i < len(c.lines); c.i++ {
		line := strings.ToLower(strings.TrimSpace(
			strings.SplitN(c.lines[c.i], ";", 2)[0]))
		if len(line) < 11 || line[0] != '[' || line[len(line)-1] != ']' ||
			line[1:10] != "statedef " {
			continue
		}
		n := Atoi(line[11:])
		if existInThisFile[n] {
			continue
		}
		existInThisFile[n] = true
		c.i++
		is, _, err := c.parseSection(nil)
		if err != nil {
			return errmes(err)
		}
		sbc := newStateBytecode(c.playerNo)
		if err := c.stateDef(is, sbc); err != nil {
			return errmes(err)
		}
		for c.i++; c.i < len(c.lines); c.i++ {
			line := strings.ToLower(strings.TrimSpace(
				strings.SplitN(c.lines[c.i], ";", 2)[0]))
			if len(line) < 7 || line[0] != '[' || line[len(line)-1] != ']' ||
				line[1:7] != "state " {
				c.i--
				break
			}
			c.i++
			c.block = newStateBlock()
			sc := newStateControllerBase()
			var scf scFunc
			var triggerall []BytecodeExp
			allUtikiri := false
			var trigger [][]BytecodeExp
			var trexist []int8
			is, ihp, err := c.parseSection(func(name, data string) error {
				switch name {
				case "type":
					var ok bool
					scf, ok = c.scmap[data]
					if !ok {
						println(data)
						unimplemented()
					}
				case "persistent":
					if n >= 0 {
						c.block.persistent = Atoi(data)
						if c.block.persistent > 128 {
							c.block.persistent = 1
						} else if c.block.persistent != 1 {
							if c.block.persistent <= 0 {
								c.block.persistent = math.MaxInt32
							}
							c.block.persistentIndex = int32(len(sbc.ctrlsps))
							sbc.ctrlsps = append(sbc.ctrlsps, 0)
						}
					}
				case "ignorehitpause":
					c.block.ignorehitpause = Atoi(data) != 0
				case "triggerall":
					be, err := c.fullExpression(&data, VT_Bool)
					if err != nil {
						return err
					}
					if len(be) == 2 && be[0] == OC_int8 {
						if be[1] == 0 {
							allUtikiri = true
						}
					} else if !allUtikiri {
						triggerall = append(triggerall, be)
					}
				default:
					tn, ok := readDigit(name[7:])
					if !ok || tn < 1 || tn > 65536 {
						errmes(Error("トリガー名 (" + name + ") が不正です"))
					}
					if len(trigger) < int(tn) {
						trigger = append(trigger, make([][]BytecodeExp,
							int(tn)-len(trigger))...)
						trexist = append(trexist, make([]int8, int(tn)-len(trexist))...)
					}
					tn--
					be, err := c.fullExpression(&data, VT_Bool)
					if err != nil {
						if sys.ignoreMostErrors {
							_break := false
							for i := 0; i < int(tn); i++ {
								if trexist[i] == 0 {
									_break = true
									break
								}
							}
							if _break {
								break
							}
						}
						return err
					}
					if len(be) == 2 && be[0] == OC_int8 {
						if be[1] == 0 {
							trexist[tn] = -1
						} else if trexist[tn] == 0 {
							trexist[tn] = 1
						}
					} else if !allUtikiri && trexist[tn] >= 0 {
						trigger[tn] = append(trigger[tn], be)
						trexist[tn] = 1
					}
				}
				return nil
			})
			if err != nil {
				return errmes(err)
			}
			if scf == nil {
				return errmes(Error("typeが指定されていません"))
			}
			if len(trexist) == 0 || trexist[0] == 0 {
				return errmes(Error("trigger1がありません"))
			}
			var texp BytecodeExp
			for i, e := range triggerall {
				texp.append(e...)
				if i < len(triggerall)-1 {
					texp.append(OC_jz8, 0)
					texp.append(OC_pop)
				}
			}
			if allUtikiri {
				if len(texp) > 0 {
					texp.appendValue(BytecodeBool(false))
				}
			} else {
				for i, tr := range trigger {
					if trexist[i] == 0 {
						break
					}
					var te BytecodeExp
					if trexist[i] < 0 {
						te.appendValue(BytecodeBool(false))
					}
					oldlen := len(te)
					for j := len(tr) - 1; j >= 0; j-- {
						tmp := tr[j]
						if j < len(tr)-1 {
							if len(te) > int(math.MaxUint8-1) {
								tmp.appendI32Op(OC_jz, int32(len(te)+1))
							} else {
								tmp.append(OC_jz8, OpCode(len(te)+1))
							}
							tmp.append(OC_pop)
						}
						te = append(tmp, te...)
					}
					if len(te) == oldlen {
						te = nil
					}
					if len(te) == 0 {
						if trexist[i] > 0 {
							if len(texp) > 0 {
								texp.appendValue(BytecodeBool(true))
							}
							break
						}
					} else {
						texp.append(te...)
						if i < len(trigger)-1 {
							texp.append(OC_jnz8, 0)
							texp.append(OC_pop)
						}
					}
				}
			}
			c.block.trigger = texp
			sctrl, err := scf(is, sbc, sc, ihp)
			if err != nil {
				return errmes(err)
			}
			appending := true
			if len(texp) == 0 {
				if allUtikiri {
					appending = false
				} else {
					appending = false
					for _, te := range trexist {
						if te >= 0 {
							if te > 0 {
								appending = true
							}
							break
						}
					}
				}
			}
			if appending {
				if len(c.block.trigger) == 0 && c.block.persistentIndex < 0 &&
					c.block.ignorehitpause {
					sbc.block.ctrls = append(sbc.block.ctrls, sctrl)
				} else {
					c.block.ctrls = append(c.block.ctrls, sctrl)
					sbc.block.ctrls = append(sbc.block.ctrls, *c.block)
				}
			}
		}

		if _, ok := bc.states[n]; !ok {
			bc.states[n] = *sbc
		}
	}
	return nil
}
func (c *Compiler) yokisinaiToken() error {
	if c.token == "" {
		return Error("予期されていないファイル終端")
	}
	return Error("予期されていないトークン: " + c.token)
}
func (c *Compiler) nextLine() (string, bool) {
	s := <-c.linechan
	if s == nil {
		return "", false
	}
	return *s, true
}
func (c *Compiler) scan(line *string) string {
	for {
		c.token = c.tokenizer(line)
		if len(c.token) > 0 {
			if c.token[0] != '#' {
				break
			}
		}
		var ok bool
		*line, ok = c.nextLine()
		if !ok {
			break
		}
	}
	return c.token
}
func (c *Compiler) needToken(t string) error {
	if c.token != t {
		if c.token == "" {
			return Error(t + "が必要な場所で予期されていないファイル終端")
		}
		return Error(t + "が必要な場所で予期されていないトークン: " + c.token)
	}
	return nil
}
func (c *Compiler) readString(line *string) (string, error) {
	i := strings.Index(*line, "\"")
	if i < 0 {
		return "", Error("'\"' が閉じられていない")
	}
	s := (*line)[:i]
	*line = (*line)[i+1:]
	return s, nil
}
func (c *Compiler) readSentenceLine(line *string) (s string, err error) {
	c.token = ""
	offset := 0
	for {
		i := strings.IndexAny((*line)[offset:], ";#\"{}")
		if i < 0 {
			s, *line = *line, ""
			return
		}
		i += offset
		switch (*line)[i] {
		case ';', '{', '}':
			c.token = (*line)[i : i+1]
			s, *line = (*line)[:i], (*line)[i+1:]
		case '#':
			s, *line = (*line)[:i], ""
		case '"':
			tmp := (*line)[i+1:]
			if _, err := c.readString(&tmp); err != nil {
				return "", err
			}
			offset = len(*line) - len(tmp)
			continue
		}
		break
	}
	return
}
func (c *Compiler) readSentence(line *string) (s string, err error) {
	sen, err := c.readSentenceLine(line)
	if err != nil {
		return "", err
	}
	for c.token == "" {
		var ok bool
		*line, ok = c.nextLine()
		if !ok {
			break
		}
		s, err := c.readSentenceLine(line)
		if err != nil {
			return "", err
		}
		sen += "\n" + s
	}
	return strings.TrimSpace(sen), nil
}
func (c *Compiler) statementEnd(line *string) error {
	c.token = c.tokenizer(line)
	if len(c.token) > 0 && c.token[0] != '#' {
		return c.yokisinaiToken()
	}
	c.token = ""
	return nil
}
func (c *Compiler) stateBlock(line *string, bl *StateBlock, root bool) error {
	for {
		if c.token == "" {
			*line, _ = c.nextLine()
		}
		switch c.scan(line) {
		case "", "[":
			if !root {
				return c.yokisinaiToken()
			}
			return nil
		case "}":
			if root {
				return c.yokisinaiToken()
			}
			if len(bl.trigger) > 0 {
				unimplemented()
			} else if err := c.statementEnd(line); err != nil {
				return err
			}
			return nil
		case "if":
			expr, err := c.readSentence(line)
			if err != nil {
				return err
			}
			otk := c.token
			ifbl := newStateBlock()
			if ifbl.trigger, err = c.fullExpression(&expr, VT_Bool); err != nil {
				return err
			}
			c.token = otk
			if err := c.needToken("{"); err != nil {
				return err
			}
			if err := c.stateBlock(line, ifbl, false); err != nil {
				return err
			}
			bl.ctrls = append(bl.ctrls, *ifbl)
		}
		break
	}
	return c.yokisinaiToken()
}
func (c *Compiler) stateCompileZ(bc *Bytecode, filename, src string) error {
	defer func(oime bool) {
		sys.ignoreMostErrors = oime
	}(sys.ignoreMostErrors)
	sys.ignoreMostErrors = false
	c.block = nil
	c.lines, c.i = SplitAndTrim(src, "\n"), 0
	c.linechan = make(chan *string)
	endchan := make(chan bool)
	end := false
	defer func() {
		if !end {
			endchan <- true
		}
	}()
	go func() {
		var sp *string
		i := c.i
		if i < len(c.lines) {
			str := strings.TrimSpace(c.lines[i])
			sp = &str
		}
		for {
			select {
			case <-endchan:
				close(c.linechan)
				return
			case c.linechan <- sp:
				if i < len(c.lines) {
					c.i = i
					i++
					str := strings.TrimSpace(c.lines[i])
					sp = &str
				} else {
					sp = nil
				}
			}
		}
	}()
	errmes := func(err error) error {
		endchan <- true
		end = true
		return Error(fmt.Sprintf("%v:%v:\n%v", filename, c.i+1, err.Error()))
	}
	existInThisFile := make(map[int32]bool)
	var line string
	c.token = ""
	for {
		if c.token == "" {
			var ok bool
			line, ok = c.nextLine()
			if !ok {
				break
			}
			if len(line) == 0 {
				continue
			}
			c.scan(&line)
		}
		if c.token != "[" {
			if c.token[0] != '#' {
				return c.yokisinaiToken()
			}
			c.token = ""
			continue
		}
		switch c.scan(&line) {
		case "":
			return c.yokisinaiToken()
		case "statedef":
			c.scan(&line)
			n, err := c.integer2(&line)
			if err != nil {
				return errmes(err)
			}
			if existInThisFile[n] {
				return errmes(Error(fmt.Sprintf("State %v の多重定義", n)))
			}
			existInThisFile[n] = true
			is := NewIniSection()
			for c.token != "]" {
				switch c.token {
				case ";":
					name := c.scan(&line)
					if name == "" {
						return errmes(c.yokisinaiToken())
					}
					if name == "]" {
						break
					}
					c.scan(&line)
					if err := c.needToken(":"); err != nil {
						return errmes(err)
					}
					data, err := c.readSentence(&line)
					if err != nil {
						return errmes(err)
					}
					is[name] = data
				default:
					return errmes(c.yokisinaiToken())
				}
			}
			sbc := newStateBytecode(c.playerNo)
			if err := c.stateDef(is, sbc); err != nil {
				return errmes(err)
			}
			if err := c.statementEnd(&line); err != nil {
				return errmes(err)
			}
			if err := c.stateBlock(&line, &sbc.block, true); err != nil {
				return errmes(err)
			}
			if _, ok := bc.states[n]; !ok {
				bc.states[n] = *sbc
			}
		default:
			return errmes(Error("認識できないセクション名: " + c.token))
		}
	}
	return nil
}
func (c *Compiler) Compile(pn int, def string) (*Bytecode, error) {
	c.playerNo = pn
	bc := newBytecode()
	str, err := LoadText(def)
	if err != nil {
		return nil, err
	}
	lines, i, cmd, stcommon := SplitAndTrim(str, "\n"), 0, "", ""
	var st [11]string
	info, files := true, true
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		switch name {
		case "info":
			if info {
				info = false
				sys.cgi[pn].ver = [2]uint16{0, 0}
				str, ok := is["mugenversion"]
				if ok {
					for i, s := range SplitAndTrim(str, ".") {
						if i >= len(sys.cgi[pn].ver) {
							break
						}
						if v, err := strconv.ParseUint(s, 10, 16); err == nil {
							sys.cgi[pn].ver[i] = uint16(v)
						} else {
							break
						}
					}
				}
			}
		case "files":
			if files {
				files = false
				cmd, stcommon = is["cmd"], is["stcommon"]
				st[0] = is["st"]
				for i := 1; i < len(st); i++ {
					st[i] = is[fmt.Sprintf("st%d", i-1)]
				}
			}
		}
	}
	if err := LoadFile(&cmd, def, func(filename string) error {
		str, err := LoadText(filename)
		if err != nil {
			return err
		}
		lines, i = SplitAndTrim(str, "\n"), 0
		return nil
	}); err != nil {
		return nil, err
	}
	if sys.chars[pn][0].cmd == nil {
		sys.chars[pn][0].cmd = make([]CommandList, MaxSimul*2)
		b := newCommandBuffer()
		for i := range sys.chars[pn][0].cmd {
			sys.chars[pn][0].cmd[i] = *NewCommandList(b)
		}
	}
	c.cmdl = &sys.chars[pn][0].cmd[pn]
	remap, defaults, ckr := true, true, NewCommandKeyRemap()
	var cmds []IniSection
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		switch name {
		case "remap":
			if remap {
				remap = false
				rm := func(name string, k, nk *CommandKey) {
					switch strings.ToLower(is[name]) {
					case "x":
						*k, *nk = CK_x, CK_nx
					case "y":
						*k, *nk = CK_y, CK_ny
					case "z":
						*k, *nk = CK_z, CK_nz
					case "a":
						*k, *nk = CK_a, CK_na
					case "b":
						*k, *nk = CK_b, CK_nb
					case "c":
						*k, *nk = CK_c, CK_nc
					case "s":
						*k, *nk = CK_s, CK_ns
					}
				}
				rm("x", &ckr.x, &ckr.nx)
				rm("y", &ckr.y, &ckr.ny)
				rm("z", &ckr.z, &ckr.nz)
				rm("a", &ckr.a, &ckr.na)
				rm("b", &ckr.b, &ckr.nb)
				rm("c", &ckr.c, &ckr.nc)
				rm("s", &ckr.s, &ckr.ns)
			}
		case "defaults":
			if defaults {
				defaults = false
				is.ReadI32("command.time", &c.cmdl.DefaultTime)
				var i32 int32
				if is.ReadI32("command.buffer.time", &i32) {
					c.cmdl.DefaultBufferTime = Max(1, i32)
				}
			}
		default:
			if len(name) >= 7 && name[:7] == "command" {
				cmds = append(cmds, is)
			}
		}
	}
	for _, is := range cmds {
		name, _, err := is.getText("name")
		if err != nil {
			return nil, Error(fmt.Sprintf("%v:\nname: %v\n%v",
				cmd, name, err.Error()))
		}
		cm, err := ReadCommand(name, is["command"], ckr)
		if err != nil {
			return nil, Error(cmd + ":\nname = " + is["name"] +
				"\ncommand = " + is["command"] + "\n" + err.Error())
		}
		is.ReadI32("time", &cm.time)
		var i32 int32
		if is.ReadI32("buffer.time", &i32) {
			cm.buftime = Max(1, i32)
		}
		c.cmdl.Add(*cm)
	}
	sys.stringPool[pn].Clear()

	// test
	if err := c.stateCompile(bc, "common1.cns", def); err != nil {
		return nil, err
	}

	for _, s := range st {
		if len(s) > 0 {
			if err := c.stateCompile(bc, s, def); err != nil {
				return nil, err
			}
		}
	}
	if err := c.stateCompile(bc, cmd, def); err != nil {
		return nil, err
	}
	if len(stcommon) > 0 {
		if err := c.stateCompile(bc, stcommon, def); err != nil {
			return nil, err
		}
	}
	return bc, nil
}
