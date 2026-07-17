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
  // Longdao AI home page uses home.* namespace: merge upstream landing.home with Longdao custom, Longdao wins
  home: { ...(landing as Record<string, unknown>).home as object, ...homeLongdao },
}
