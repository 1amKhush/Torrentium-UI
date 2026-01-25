import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import Home from './views/Home.vue'
import FileDetails from './views/FileDetails.vue'
import './style.css'

const routes = [
  { path: '/', component: Home },
  { path: '/file/:cid', component: FileDetails },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

const app = createApp(App)
app.use(router)
app.mount('#app')
