package ai

import "strings"

const basePrompt = `You are editing a real product photo for a resale marketplace listing.

Hard rules (must follow):

* Do NOT change the product itself in any way: do not alter shape, size, color, logo, text, material texture, patterns, stains, scratches, dents, wear, or any defects.
* Do NOT add, remove, or hallucinate any parts of the product or accessories.
* Do NOT "beautify" the product surface. Preserve every mark and imperfection exactly as-is.
* Keep the product in the same position and perspective (no rotation or re-framing that changes its appearance).
* Do NOT change the composition: keep the framing, orientation, and product scale identical. Do not crop or zoom differently.
* Only edit the environment: background/desk/floor/walls, lighting, shadows, and remove unrelated clutter around the product.
* Output must look photorealistic, like a product-style photo shot in natural soft daylight.

Task:
Clean up the background and lighting while preserving the product perfectly unchanged.`

var stylePrompts = map[string]string{
	"fashion-look": `Style target (fashion-look):

* Create a clean white seamless background (pure white to very light warm white).

* Soft daylight, minimal natural shadows directly under the product.

* Remove any floor stains, wrinkles, dust, clutter, and surrounding objects.

* Keep the product edges crisp and true to the original; do not alter fabric texture or color.

* Overall look: product-style photo of a folded garment on a clean white background, soft daylight, minimal shadows.`,
	"tech-gadget": `Style target (tech-gadget):

* Place the unchanged product on a light wooden desk surface (subtle natural grain, modern airy aesthetic).

* Airy soft lighting, neutral white balance, gentle shadowing.

* Remove any clutter, cables, stains, and distracting background items.

* Keep the product perfectly unchanged (no surface smoothing, no color shift).

* Overall look: sleek flatlay on a light wooden desk, airy lighting, modern aesthetic.`,
	"outdoor-gear": `Style target (outdoor-gear):

* Place the unchanged product on rustic wood (slightly darker, outdoor-feel, but still clean).

* Warm daylight, crisp focus, natural shadowing.

* Remove clutter, dirt, stains, and unrelated objects; keep it neat and intentional.

* Preserve the product exactly as-is (wear/marks must remain).

* Overall look: neatly arranged gear on rustic wood, warm daylight, crisp focus.`,
}

const safetySuffix = `If the product surface looks cleaner than the input, revert: the product must be identical to the input photo. Only the background and lighting may change.
Do not alter composition, orientation, or scale. Background should be clean white; product remains untouched.
No text, no watermark, no added props.`

// BuildEnhancePrompt concatenates base, style, and safety prompts. Mode fallback defaults to fashion-look.
func BuildEnhancePrompt(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	style, ok := stylePrompts[mode]
	if !ok {
		style = stylePrompts["fashion-look"]
	}
	parts := []string{basePrompt, style, safetySuffix}
	return strings.Join(parts, "\n\n")
}
