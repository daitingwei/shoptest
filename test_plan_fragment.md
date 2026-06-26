import (
    // ...
    "github.com/go-kratos/kratos/contrib/encoding/form/v2"  // 引入 form 编解码器
)

func NewHTTPServer(...) *http.Server {
    var opts = []http.ServerOption{
        http.Middleware(
            recovery.Recovery(),
        ),
        // 不需要额外配，只要 import 了这个包，
        // kratos 会自动注册表单和 multipart 支持
    }
    // 或者更显式的方式：
    _ = form.Name  // 保证 init() 被执行
}
