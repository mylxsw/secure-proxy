package handler

import (
	"fmt"
	"net/http"

	"github.com/mylxsw/secure-proxy/internal/template"
)

func (handler *AuthHandler) buildStatusHandler() func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html")

		userAuthInfo, err := handler.cookieManager.GetCookie(r)
		if err != nil {
			http.Redirect(rw, r, "/secure-proxy/auth", http.StatusSeeOther)
			return
		}

		if err := template.Render(statusPage, rw, userAuthInfo); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}
	}
}

var statusPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>统一身份认证 - 状态</title>

    <link href="/secure-proxy/assets/css/bootstrap.min.css" rel="stylesheet">

    <style>
      .bd-placeholder-img {
        font-size: 1.125rem;
        text-anchor: middle;
        -webkit-user-select: none;
        -moz-user-select: none;
        user-select: none;
      }

      @media (min-width: 768px) {
        .bd-placeholder-img-lg {
          font-size: 3.5rem;
        }
      }
    </style>

    <link href="/secure-proxy/assets/style.css?v=20220118" rel="stylesheet">
  </head>
  <body class="text-center">
    
  <main class="form-signin">
	<div class="card card-default">
		<div class="card-header">状态</div>
		<div class="card-body text-left">
			<p>当前登录信息：</p>
			<li>用户：<b>{{ .Account }}({{ .Name }})</b></li>
			<li>登录时间：<b>{{ .CreatedAt }}</b></li>
			<li>用户组：
			{{- range $i, $group := .Groups }}
			<span class="badge rounded-pill bg-success">{{ $group }}</span>
			{{- end }}
			</li>
			<p class="mt-2"><a href="/secure-proxy/auth/logout" class="btn btn-primary">退出</a></p>
		</div>
		<p class="mt-1 mb-3 text-muted">© 2022</p>
	</div>
  </main>

<script>

</script>
    
  </body>
</html>`
