package house

import (
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/glop/util/algorithm"
  "github.com/runningwild/haunts/base"
)

type FurniturePanel struct {
  *gui.VerticalTable
  name       *gui.TextEditLine
  room_size  *gui.ComboBox
  floor_path *gui.FileWidget
  wall_path  *gui.FileWidget

  Room       *roomDef
  RoomViewer *RoomViewer

  // If we're in the middle of moving an object and this widget gets collapsed
  // we want to put the object back where it was before we started dragging it.
  prev_object *Furniture

  // Distance from the mouse to the center of the object, in board coordinates
  drag_anchor struct{ x,y float32 }

  // True iff the selected object should be placed when the mouse button is
  // released.  If false this object will be placed when the mouse button is
  // clicked.
  drop_on_release bool

  key_map base.KeyMap
}

func (w *FurniturePanel) Collapse() {
  w.Room.Furniture = algorithm.Choose(w.Room.Furniture, func(a interface{}) bool {
    return a.(*Furniture) != w.prev_object
  }).([]*Furniture)
  w.prev_object = nil
  w.RoomViewer.Temp.Furniture = nil
}
func (w *FurniturePanel) Expand() {
  w.RoomViewer.SetEditMode(editFurniture)
}

func makeFurniturePanel(room *roomDef, viewer *RoomViewer) *FurniturePanel {
  var fp FurniturePanel
  fp.Room = room
  fp.RoomViewer = viewer
  fp.key_map = base.GetDefaultKeyMap()
  if room.Name == "" {
    room.Name = "name"
  }
  fp.name = gui.MakeTextEditLine("standard", room.Name, 300, 1, 1, 1, 1)  

  if room.Floor.Path == "" {
    room.Floor.Path = base.Path(datadir)
  }
  fp.floor_path = gui.MakeFileWidget(room.Floor.Path.String(), imagePathFilter)

  if room.Wall.Path == "" {
    room.Wall.Path = base.Path(datadir)
  }
  fp.wall_path = gui.MakeFileWidget(room.Wall.Path.String(), imagePathFilter)

  fp.room_size = gui.MakeComboTextBox(algorithm.Map(tags.RoomSizes, []string{}, func(a interface{}) interface{} { return a.(RoomSize).String() }).([]string), 300)
  for i := range tags.RoomSizes {
    if tags.RoomSizes[i].String() == room.Size.String() {
      fp.room_size.SetSelectedIndex(i)
      break
    }
  }
  fp.VerticalTable = gui.MakeVerticalTable()
  fp.VerticalTable.Params().Spacing = 3  
  fp.VerticalTable.Params().Background.R = 0.3
  fp.VerticalTable.Params().Background.B = 1
  fp.VerticalTable.AddChild(fp.name)
  fp.VerticalTable.AddChild(fp.floor_path)
  fp.VerticalTable.AddChild(fp.wall_path)
  fp.VerticalTable.AddChild(fp.room_size)

  furn_table := gui.MakeVerticalTable()
  fnames := GetAllFurnitureNames()
  for i := range fnames {
    name := fnames[i]
    furn_table.AddChild(gui.MakeButton("standard", name, 300, 1, 1, 1, 1, func(t int64) {
      f := MakeFurniture(name)
      if f == nil { return }
      fp.RoomViewer.Temp.Furniture = f
      fp.drop_on_release = false
      dx,dy := fp.RoomViewer.Temp.Furniture.Dims()
      fp.drag_anchor.x = float32(dx - 1) / 2
      fp.drag_anchor.y = float32(dy - 1) / 2
    }))
  }
  fp.VerticalTable.AddChild(gui.MakeScrollFrame(furn_table, 300, 600))

  return &fp
}

func (w *FurniturePanel) Respond(ui *gui.Gui, group gui.EventGroup) bool {
  if w.VerticalTable.Respond(ui, group) {
    return true
  }
  if found,event := group.FindEvent(gin.Escape); found && event.Type == gin.Press {
    w.RoomViewer.Temp.Furniture = nil
    return true
  }
  if found,event := group.FindEvent(w.key_map["rotate left"].Id()); found && event.Type == gin.Press {
    if w.RoomViewer.Temp.Furniture != nil {
      w.RoomViewer.Temp.Furniture.RotateLeft()
    }
  }
  if found,event := group.FindEvent(w.key_map["rotate right"].Id()); found && event.Type == gin.Press {
    if w.RoomViewer.Temp.Furniture != nil {
      w.RoomViewer.Temp.Furniture.RotateRight()
    }
  }
  if found,event := group.FindEvent(gin.MouseLButton); found {
    if w.RoomViewer.Temp.Furniture != nil && (event.Type == gin.Press || (event.Type == gin.Release && w.drop_on_release)) {
      fx := w.RoomViewer.Temp.Furniture.X
      fy := w.RoomViewer.Temp.Furniture.Y
      fdx,fdy := w.RoomViewer.Temp.Furniture.Dims()

      if fx >= 0 && fy >= 0 && fx + fdx <= w.RoomViewer.room.Size.Dx && fy + fdy <= w.RoomViewer.room.Size.Dy {
        w.Room.Furniture = append(w.Room.Furniture, w.RoomViewer.Temp.Furniture)
      }
      w.RoomViewer.Temp.Furniture = nil
    } else if w.RoomViewer.Temp.Furniture == nil && event.Type == gin.Press {
      bx,by := w.RoomViewer.WindowToBoard(event.Key.Cursor().Point())
      w.RoomViewer.Temp.Furniture = nil
      for i := range w.Room.Furniture {
        x,y := w.Room.Furniture[i].Pos()
        dx,dy := w.Room.Furniture[i].Dims()
        if int(bx) >= x && int(bx) < x + dx && int(by) >= y && int(by) < y + dy {
          w.RoomViewer.Temp.Furniture = w.Room.Furniture[i]
          w.Room.Furniture[i] = w.Room.Furniture[len(w.Room.Furniture) - 1]
          w.Room.Furniture = w.Room.Furniture[0 : len(w.Room.Furniture) - 1]
          break
        }
      }
      if w.RoomViewer.Temp.Furniture != nil {
        w.prev_object = new(Furniture)
        *w.prev_object = *w.RoomViewer.Temp.Furniture
      }
      w.Room.Furniture = algorithm.Choose(w.Room.Furniture, func(a interface{}) bool {
        return a.(*Furniture) != w.RoomViewer.Temp.Furniture
      }).([]*Furniture)
      w.drop_on_release = true
      if w.RoomViewer.Temp.Furniture != nil {
        px,py := w.RoomViewer.Temp.Furniture.Pos()
        w.drag_anchor.x = bx - float32(px) - 0.5
        w.drag_anchor.y = by - float32(py) - 0.5
      }
    }
    return true
  }
  return false
}

func (w *FurniturePanel) Reload() {
  for i := range tags.RoomSizes {
    if tags.RoomSizes[i].String() == w.Room.Size.String() {
      w.room_size.SetSelectedIndex(i)
      break
    }
  }
  w.name.SetText(w.Room.Name)
  w.floor_path.SetPath(w.Room.Floor.Path.String())
  w.wall_path.SetPath(w.Room.Wall.Path.String())
}

func (w *FurniturePanel) Think(ui *gui.Gui, t int64) {
  if w.RoomViewer.Temp.Furniture != nil {
    mx,my := gin.In().GetCursor("Mouse").Point()
    bx,by := w.RoomViewer.WindowToBoard(mx, my)
    w.RoomViewer.Temp.Furniture.X = int(bx - w.drag_anchor.x)
    w.RoomViewer.Temp.Furniture.Y = int(by - w.drag_anchor.y)
  }

  w.VerticalTable.Think(ui, t)
  w.Room.Resize(tags.RoomSizes[w.room_size.GetComboedIndex()])
  w.Room.Name = w.name.GetText()
  w.Room.Floor.Path = base.Path(w.floor_path.GetPath())
  w.Room.Wall.Path = base.Path(w.wall_path.GetPath())
}

