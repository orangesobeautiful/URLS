import { boot } from "quasar/wrappers";

let toRouter = null;

class ToRouter {
  constructor(router) {
    this.router = router;
  }

  // 跳轉到指定頁面
  Link(url) {
    this.router.push(url);
  }

  // 跳轉至首頁
  HomePage() {
    this.router.push("/");
  }

  // 跳轉至 登入頁面
  SigninPage() {
    this.router.push("/signin");
  }

  // 跳轉至 註冊頁面
  RegisterPage() {
    this.router.push("/register");
  }

  // 跳轉至 網頁控制台頁面
  DashboardPage() {
    this.router.push("/dashboard/settings");
  }

  // 跳轉至 使用者圖片頁面
  SelfLinksPage() {
    this.router.push("/self-links");
  }

  // 重新整理
  Reload() {
    this.router.go(0);
  }

  // 上一頁
  PreviousPage(ignoreList) {
    let toBack = true;
    const current = this.router.options.history.state.current;
    const back = this.router.options.history.state.back;
    if (back != null && back.startsWith("/") && current != back) {
      for (const ignore of ignoreList) {
        if (back == ignore) {
          toBack = false;
          break;
        }
      }
    } else {
      toBack = false;
    }

    if (toBack) {
      this.router.push(back);
    } else {
      this.router.push("/");
    }
  }
}

// "async" is optional;
// more info on params: https://v2.quasar.dev/quasar-cli/boot-files
export default boot(async ({ router }) => {
  toRouter = new ToRouter(router);
});

export { toRouter };
