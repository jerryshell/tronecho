// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: ["@nuxt/ui", "@vueuse/nuxt"],

  devtools: {
    enabled: true,
  },

  css: ["~/assets/css/main.css"],

  compatibilityDate: "2026-07-08",

  routeRules: {
    "/api/**": {
      cors: true,
    },
  },

  ui: {
    fonts: false,
  },

  runtimeConfig: {
    natsUrl: process.env.NATS_URL || "nats://localhost:4222",
    tronechoPrefix: process.env.TRONECHO_PREFIX || "tronecho",
  },

  nitro: {
    storage: {
      customers: {
        driver: "fs",
        base: "./data/customers",
      },
      deposits: {
        driver: "fs",
        base: "./data/deposits",
      },
    },
  },
});
