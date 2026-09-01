import type { main } from '../wailsjs/go/models'
import {
  AddFiles,
  AddPaths,
  ChangeIcon,
  GetData,
  Launch,
  Reveal,
  SaveData,
  Stop,
  UpdateIcon,
} from '../wailsjs/go/main/App'

export type AppItem = main.AppItem
export type CategoryNode = main.CategoryNode
export type AppStore = main.AppStore
export type AppData = main.AppData
export type ItemState = main.ItemState
export type AddResult = main.AddResult
export type IconResult = main.IconResult

export {
  AddFiles,
  AddPaths,
  ChangeIcon,
  GetData,
  Launch,
  Reveal,
  SaveData,
  Stop,
  UpdateIcon,
}
