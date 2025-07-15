package parsers

type Tag struct {
	Value string `sg:"required,delimiter=,"`
}

func (t Tag) Name() string    { return "TAG" }
func (t Tag) Mode() ParseMode { return ModePositional }

type Security struct {
	Value   string   `sg:"required"`
	Exclude []string `sg:"delimiter=,"` // 支持,分割和默认支持空格分割
	Include []string `sg:"delimiter=,"`
}

func (s Security) Name() string    { return "SECURITY" }
func (s Security) Mode() ParseMode { return ModeNamed }

type Header struct {
	Value       string `sg:"required"`
	Required    bool   `sg:"required"`
	Description string
}

func (s Header) Name() string    { return "HEADER" }
func (s Header) Mode() ParseMode { return ModePositional }

/////////////////////////////// 响应 /////////////////////////////////////

type JSON struct {
}

func (s JSON) Name() string    { return "JSON" }
func (s JSON) Mode() ParseMode { return ModePositional }

type MIME struct {
	// Alias	MIME Type
	//json	application/json
	//xml	text/xml
	//plain	text/plain
	//html	text/html
	//mpfd	multipart/form-data
	//x-www-form-urlencoded	application/x-www-form-urlencoded
	//json-api	application/vnd.api+json
	//json-stream	application/x-json-stream
	//octet-stream	application/octet-stream
	//png	image/png
	//jpeg	image/jpeg
	//gif	image/gif
	//event-stream	text/event-stream
	Value string `sg:"required"`
}

func (s MIME) Name() string    { return "MIME" }
func (s MIME) Mode() ParseMode { return ModePositional }

/////////////////////////////// 请求 /////////////////////////////////////

type FormReq struct{}

func (s FormReq) Name() string    { return "FORM-REQ" }
func (s FormReq) Mode() ParseMode { return ModePositional }

type JsonReq struct{}

func (s JsonReq) Name() string    { return "JSON-REQ" }
func (s JsonReq) Mode() ParseMode { return ModePositional }

/////////////////////////////// GIN-Handler /////////////////////////////////////

type MiddleWare struct {
	Value []string `sg:"required"`
}

func (s MiddleWare) Name() string    { return "MID" }
func (s MiddleWare) Mode() ParseMode { return ModePositional }

/////////////////////// GET|PUT|POST|PATCH|DELETE ///////////////////////

type GET struct {
	Value string `sg:"required,delimiter=,"`
}

func (s GET) Name() string    { return "GET" }
func (s GET) Mode() ParseMode { return ModePositional }

type POST struct {
	Value string `sg:"required"`
}

func (s POST) Name() string    { return "POST" }
func (s POST) Mode() ParseMode { return ModePositional }

type PUT struct {
	Value string `sg:"required"`
}

func (s PUT) Name() string    { return "PUT" }
func (s PUT) Mode() ParseMode { return ModePositional }

type PATCH struct {
	Value string `sg:"required"`
}

func (s PATCH) Name() string    { return "PATCH" }
func (s PATCH) Mode() ParseMode { return ModePositional }

type DELETE struct {
	Value string `sg:"required"`
}

func (s DELETE) Name() string    { return "DELETE" }
func (s DELETE) Mode() ParseMode { return ModePositional }
