package main

// 配对面板：注入到应用界面里的一个入口（设置侧栏「Pairing」），
// 面板提供：局域网直连开关、服务器地址、访问令牌、一次性配对码 + 二维码。
// 与 opencode 前端版本弱耦合（按按钮文本定位，找不到就静默不注入）。
const PAIR_UI = `;(function(){
  try{
    var PB="/app/opencode";
    if(location.pathname.indexOf("/")!==0)return;
    var KIND="Pairing";

    function api(path,opt){
      return fetch(PB+path,opt||{}).then(function(r){return r.json();});
    }

    var host=null,sr=null;
    function ensureHost(){
      if(host&&document.body.contains(host))return;
      host=document.createElement("div");
      host.id="__oc_pair_host__";
      host.style.cssText="position:fixed;inset:0;z-index:2147483000;display:none";
      document.body.appendChild(host);
      sr=host.attachShadow?host.attachShadow({mode:"open"}):host;
      sr.innerHTML=[
        '<style>',
        ':host{all:initial}',
        '*{box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif}',
        '.scrim{position:fixed;inset:0;background:rgba(9,12,20,.55);display:flex;align-items:center;justify-content:center;padding:24px}',
        '.sheet{width:520px;max-width:96vw;max-height:88vh;overflow:auto;background:#fff;color:#14161a;border-radius:16px;',
        'box-shadow:0 24px 60px rgba(0,0,0,.35);padding:22px 22px 18px}',
        '@media(prefers-color-scheme:dark){.sheet{background:#1b1d23;color:#e6e8ee}.row{border-color:#2a2d35!important}',
        '.sub{color:#9aa0ab!important}.val{background:#14161b!important;border-color:#2a2d35!important;color:#cfd4dc!important}',
        '.btn{background:#2a2d35!important;color:#e6e8ee!important}.btn.main{background:#5eead4!important;color:#0b1020!important}}',
        '.hd{display:flex;align-items:center;justify-content:space-between;margin-bottom:6px}',
        'h2{font-size:16px;margin:0;font-weight:650}',
        '.x{border:0;background:transparent;font-size:22px;line-height:1;color:#8b93a1;cursor:pointer;padding:4px 8px;border-radius:8px}',
        '.x:hover{background:rgba(128,128,128,.15)}',
        '.sub{font-size:12px;color:#6b7280;line-height:1.6;margin:0 0 14px}',
        '.row{display:flex;align-items:flex-start;gap:12px;padding:12px 0;border-top:1px solid #eceef2}',
        '.meta{flex:1;min-width:0}',
        '.title{font-size:13px;font-weight:600;margin-bottom:3px}',
        '.desc{font-size:11.5px;color:#8b93a1;line-height:1.55}',
        '.val{margin-top:7px;font:12px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;background:#f5f6f8;border:1px solid #e6e8ec;',
        'border-radius:8px;padding:6px 8px;word-break:break-all;color:#374151}',
        '.act{display:flex;gap:8px;flex-shrink:0}',
        '.btn{height:30px;padding:0 12px;border-radius:8px;border:1px solid transparent;background:#f0f1f4;color:#374151;',
        'font-size:12px;cursor:pointer;white-space:nowrap}',
        '.btn:hover{filter:brightness(.96)}.btn:disabled{opacity:.55;cursor:default}',
        '.btn.main{background:#111827;color:#fff}',
        '.qr{display:none;margin-top:10px;text-align:center}',
        '.qr img{width:188px;height:188px;border-radius:10px;background:#fff;padding:6px;border:1px solid #e6e8ec}',
        '.link{font:11px/1.5 ui-monospace,Menlo,monospace;color:#4b5563;word-break:break-all;margin-top:6px}',
        '.timer{font-size:11.5px;color:#b45309;margin-top:6px}',
        '.err{font-size:11.5px;color:#dc2626;margin-top:6px}',
        '.foot{font-size:11px;color:#9ca3af;line-height:1.6;margin-top:14px}',
        '</style>',
        '<div class="scrim"><div class="sheet">',
        '<div class="hd"><h2>Pairing</h2><button class="x" data-a="close">&times;</button></div>',
        '<p class="sub">让其他设备连上这台 NAS 上的 OpenCode：扫码配对，或填地址 + 令牌/密码。</p>',
        '<div class="row"><div class="meta"><div class="title">局域网直连</div>',
        '<div class="desc" data-el="lanDesc">未开启。开启后监听端口，供桌面端和手机 App 连接。</div></div>',
        '<div class="act"><button class="btn" data-a="toggle">开启</button></div></div>',
        '<div class="row"><div class="meta"><div class="title">服务器地址</div>',
        '<div class="val" data-el="addr">—</div></div>',
        '<div class="act"><button class="btn" data-a="copyAddr">复制</button></div></div>',
        '<div class="row"><div class="meta"><div class="title">访问令牌</div>',
        '<div class="val" data-el="token">••••••••••••</div></div>',
        '<div class="act"><button class="btn" data-a="showToken">显示</button><button class="btn" data-a="copyToken">复制</button></div></div>',
        '<div class="row"><div class="meta"><div class="title">配对码</div>',
        '<div class="desc">一次性，5 分钟内有效。桌面端选「添加服务器 → 粘贴配对链接」，手机 App 直接扫码。</div>',
        '<div class="qr" data-el="qr"><img alt="配对二维码"></div>',
        '<div class="link" data-el="link" style="display:none"></div>',
        '<div class="timer" data-el="timer" style="display:none"></div>',
        '<div class="err" data-el="err" style="display:none"></div>',
        '</div><div class="act"><button class="btn main" data-a="pair">生成</button></div></div>',
        '<p class="foot">直连端口走令牌鉴权：浏览器首次访问需输入令牌；客户端可用「地址 + 密码」直接连接。不需要时把上面的开关关掉即可。</p>',
        '</div></div>'
      ].join("");
      sr.querySelector(".scrim").addEventListener("click",function(e){
        if(e.target&&e.target.className==="scrim")close();
      });
      sr.addEventListener("click",onClick,true);
    }

    var state={enabled:false,port:"",addr:"",token:"",shown:false,timer:null};

    function q(sel){return sr.querySelector(sel);}
    function setText(sel,txt){var e=q(sel);if(e)e.textContent=txt;}

    function paint(s){
      state.enabled=!!s.enabled;state.port=s.port||"";
      var urls=(s.urls&&s.urls.length)?s.urls:[];
      state.addr=urls[0]||"";
      state.token=s.token||"";
      setText('[data-el=addr]',state.addr||"—");
      setText('[data-el=token]',state.shown?(state.token||"—"):"••••••••••••");
      var t=q('[data-a=toggle]');if(t)t.textContent=state.enabled?"关闭":"开启";
      setText('[data-el=lanDesc]',state.enabled
        ?"已开启，监听 "+state.port+" 端口，其他设备可连接。"
        :"未开启。开启后监听端口，供桌面端和手机 App 连接。");
      var pb=q('[data-a=pair]');if(pb)pb.disabled=!state.enabled;
    }

    function loadState(){
      return api("/__pair/api/state").then(paint).catch(function(e){
        setText('[data-el=lanDesc]',"读取状态失败：网络不可用");
      });
    }

    function onClick(e){
      var t=e.target,b=t;
      while(b&&b!==sr&&!(b.getAttribute&&b.getAttribute("data-a")))b=b.parentNode;
      if(!b||b===sr)return;
      var a=b.getAttribute("data-a");
      e.stopPropagation();
      if(a==="close"){close();return;}
      if(a==="toggle"){toggle(b);return;}
      if(a==="copyAddr"){copy(state.addr,b);return;}
      if(a==="showToken"){state.shown=!state.shown;setText('[data-el=token]',state.shown?state.token:"••••••••••••");b.textContent=state.shown?"隐藏":"显示";return;}
      if(a==="copyToken"){copy(state.token,b);return;}
      if(a==="pair"){genPair(b);return;}
    }

    function copy(txt,btn){
      if(!txt)return;
      try{navigator.clipboard.writeText(txt);}catch(e){}
      var old=btn.textContent;btn.textContent="已复制";
      setTimeout(function(){btn.textContent=old;},1200);
    }

    function toggle(btn){
      btn.disabled=true;
      var on=!state.enabled;
      api("/__pair/api/enable",{method:"POST",headers:{"Content-Type":"application/json"},
        body:JSON.stringify({on:on,port:state.port})}).then(function(j){
        btn.disabled=false;
        if(j.error){setText('[data-el=lanDesc]',j.error);return;}
        loadState();
      }).catch(function(){btn.disabled=false;});
    }

    function genPair(btn){
      btn.disabled=true;btn.textContent="生成中…";
      setText('[data-el=err]',"");q('[data-el=err]').style.display="none";
      api("/__pair/api/pair",{method:"POST"}).then(function(j){
        btn.disabled=false;btn.textContent="重新生成";
        if(j.error){var e=q('[data-el=err]');e.style.display="block";e.textContent=j.error;return;}
        var box=q('[data-el=qr]');box.style.display="block";
        box.querySelector("img").src=j.qr||"";
        var lk=q('[data-el=link]');lk.style.display="block";lk.textContent=j.link||"";
        var tm=q('[data-el=timer]');tm.style.display="block";
        var left=j.expires_in||300;
        if(state.timer)clearInterval(state.timer);
        state.timer=setInterval(function(){
          left--;
          if(left<=0){clearInterval(state.timer);tm.textContent="已过期，请重新生成";return;}
          var m=Math.floor(left/60),s=left%60;
          tm.textContent="剩余 "+m+":"+(s<10?"0":"")+s;
        },1000);
        tm.textContent="剩余 "+Math.floor(left/60)+":"+(left%60<10?"0":"")+(left%60);
      }).catch(function(){
        btn.disabled=false;btn.textContent="生成";
        var e=q('[data-el=err]');e.style.display="block";e.textContent="请求失败，请稍后重试";
      });
    }

    function open(){
      ensureHost();
      host.style.display="block";
      host.style.pointerEvents="auto";
      loadState();
    }
    function close(){
      if(host)host.style.display="none";
    }

    // ---- 设置侧栏注入入口 ----
    var nav=null;
    function mountNav(){
      if(nav&&document.contains(nav))return;
      var bs=document.querySelectorAll("button,[role=button]");
      var anchor=null;
      for(var i=0;i<bs.length;i++){
        var t=(bs[i].innerText||"").trim();
        var r=bs[i].getBoundingClientRect();
        if(t==="设置"&&r.width>0&&r.height>0){anchor=bs[i];break;}
      }
      if(!anchor)return;
      var n=anchor.cloneNode(true);
      n.removeAttribute("href");n.removeAttribute("data-active");
      n.id="__oc_pair_nav__";
      n.style.cursor="pointer";
      // 只改最长的文本节点，保留图标
      var w=document.createTreeWalker(n,NodeFilter.SHOW_TEXT,null),best=null,x;
      while((x=w.nextNode())){var v=(x.nodeValue||"").trim();if(v&&(!best||v.length>best.nodeValue.trim().length))best=x;}
      if(best)best.nodeValue=KIND;else n.textContent=KIND;
      n.addEventListener("click",function(e){
        e.preventDefault();e.stopPropagation();e.stopImmediatePropagation();open();
      },true);
      anchor.parentNode.insertBefore(n,anchor.nextSibling);
      nav=n;
    }
    function boot(){ensureHost();mountNav();}
    if(document.body)boot();else document.addEventListener("DOMContentLoaded",boot);
    try{new MutationObserver(function(){if(!nav||!document.contains(nav))mountNav();})
      .observe(document.documentElement,{childList:true,subtree:true});}catch(e){}
  }catch(e){}
})();`
