const routes = [
  {
    path: "/",
    component: () => import("layouts/MainLayout.vue"),
    children: [
      { path: "", component: () => import("pages/IndexPage.vue") },
      { path: "self-links", component: () => import("pages/SelfLinks.vue") },
    ],
  },
  {
    path: "/signin",
    component: () => import("layouts/EmptyLayout.vue"),
    children: [{ path: "", component: () => import("pages/Signin.vue") }],
  },
  {
    path: "/register",
    component: () => import("layouts/EmptyLayout.vue"),
    children: [{ path: "", component: () => import("pages/Register.vue") }],
  },
  {
    path: "/link-error",
    component: () => import("layouts/EmptyLayout.vue"),
    children: [
      {
        path: "not-found",
        component: () => import("pages/LinkError/NotFound.vue"),
      },
      {
        path: "deleted",
        component: () => import("pages/LinkError/Deleted.vue"),
      },
    ],
  },

  // Always leave this as last one,
  // but you can also remove it
  {
    path: "/:catchAll(.*)*",
    component: () => import("pages/ErrorNotFound.vue"),
  },
];

export default routes;
