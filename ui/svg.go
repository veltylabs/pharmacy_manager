//go:build !wasm

package ui

import "webtyp.com/svg/sprite"

// boxPath es el que dibuja el icono que comparten los tres: una caja.
const boxPath = "M2 4l6-3 6 3v8l-6 3-6-3V4zm6-1.7L4.2 4.2 8 6.1l3.8-1.9L8 2.3zM3 5.4v6l4.5 2.2V7.6L3 5.4zm10 0L8.5 7.6v6L13 11.4v-6z"

// Icons aloja IconSvg para que la aplicación descubra el sprite del módulo.
type Icons struct{}

// Ejemplo: svg := icons.IconSvg()
func (m *Icons) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		sprite.Define(IconPharmacy, "0 0 16 16", sprite.Path(boxPath)),
		sprite.Define(IconProducts, "0 0 16 16", sprite.Path(boxPath)),
		sprite.Define(IconLocations, "0 0 16 16", sprite.Path(boxPath)),
	)
}
