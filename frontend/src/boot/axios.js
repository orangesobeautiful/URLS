import { boot } from "quasar/wrappers";
import { Notify } from "quasar";
import axios from "axios";

import { toRouter } from "boot/to-router";

let baseURL = ""
let withCredentials = false
if(process.env.API_Base_URL!=undefined){
  baseURL = process.env.API_Base_URL
}
if(process.env.API_WithCredentials!=undefined){
  withCredentials = process.env.API_WithCredentials
}

// Be careful when using SSR for cross-request state pollution
// due to creating a Singleton instance here;
// If any client changes this (global) instance, it might be a
// good idea to move this instance creation inside of the
// "export default () => {}" function below (which runs individually
// for each client)
const api = axios.create({
  baseURL: baseURL,
  withCredentials: withCredentials,
});

api.interceptors.response.use(
  function (response) {
    return response;
  },
  function (error) {
    if (error.response) {
      if (error.response.status == 401) {
        const url = error.response.config.url;
        if (url != "/api/user/v1/self" && url != "/api/user/v1/signin") {
          toRouter.SigninPage();
        } else {
          throw error;
        }
      } else {
        let msg = "";
        if (error.response.data["message"]) {
          msg = error.response.data["message"];
        } else {
          msg = error.response.data;
        }

        Notify.create({
          color: "negative",
          timeout: 2000,
          message: msg,
        });
      }
    } else {
      Notify.create({
        color: "negative",
        timeout: 2000,
        message: error.toString(),
      });
    }
  }
);

export default boot(({ app }) => {
  // for use inside Vue files (Options API) through this.$axios and this.$api

  app.config.globalProperties.$axios = axios;
  // ^ ^ ^ this will allow you to use this.$axios (for Vue Options API form)
  //       so you won't necessarily have to import axios in each vue file

  app.config.globalProperties.$api = api;
  // ^ ^ ^ this will allow you to use this.$api (for Vue Options API form)
  //       so you can easily perform requests against your app's API
});

export { api };
