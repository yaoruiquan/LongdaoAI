import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import admin from './admin'
import misc from './misc'
import longdao from './longdao'
import homeLongdao from './home-longdao'

export default {
  ...landing,
  ...common,
  ...dashboard,
  admin,
  ...misc,
  longdao,
  // 龙道 AI 首页用 home.* 命名空间：合并 upstream landing.home 与龙道定制，龙道优先
  home: { ...(landing as Record<string, unknown>).home as object, ...homeLongdao },
}
