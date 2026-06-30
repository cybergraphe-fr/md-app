/**
 * Redimensionnement de deux colonnes adjacentes d'un tableau d'aperçu.
 *
 * Modèle « steal from neighbour » : la somme des deux largeurs reste constante,
 * de sorte que la largeur totale du tableau (100 %) ne change pas et qu'aucun
 * scroll horizontal n'apparaît. Chaque colonne reste ≥ `min`.
 *
 * @param startW     largeur initiale de la colonne déplacée (px)
 * @param startNextW largeur initiale de la colonne voisine de droite (px)
 * @param delta      déplacement horizontal du pointeur (px, signé)
 * @param min        largeur minimale d'une colonne (px)
 */
export function resizeColumns(
  startW: number,
  startNextW: number,
  delta: number,
  min = 48
): { width: number; nextWidth: number } {
  const total = startW + startNextW;
  // Si les deux colonnes ne peuvent pas toutes deux respecter le minimum,
  // on partage l'espace à parts égales (cas dégénéré).
  if (total < min * 2) {
    const half = total / 2;
    return { width: half, nextWidth: total - half };
  }
  let width = startW + delta;
  width = Math.max(min, Math.min(total - min, width));
  return { width, nextWidth: total - width };
}
