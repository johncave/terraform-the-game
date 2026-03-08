import { createRouter, createWebHistory } from 'vue-router'
import Landing from '../views/Landing.vue'
import Dashboard from '../views/Dashboard.vue'

const routes = [
  { path: '/', component: Landing },
  { path: '/game/:id', component: Dashboard }
]

export default createRouter({
  history: createWebHistory(),
  routes
})
