"""ATURAN AKSES DOKUMEN — satu-satunya tempat aturan ini ditulis. Dipakai fitur documents dan search.

Seorang user melihat dokumen bila project-nya sama, dokumen aktif, dan
  - ia admin project / superadmin (semua status), atau
  - dokumen sudah 'ready' dan role-nya terdaftar di ai.document_roles.

Aturan ada di SQL (bukan di tampilan): dokumen yang tidak boleh dilihat tidak pernah terbaca, sehingga
tidak mungkin ikut ke hasil pencarian maupun (kelak) ke konteks LLM.

Dipakai sebagai kondisi WHERE pada tabel ai.documents beralias `d`. Parameter: project_id, is_admin, role.
"""

VISIBLE_DOCUMENTS = """
    d.project_id = %(project_id)s AND d.is_active
    AND (%(is_admin)s OR (d.processing_status = 'ready' AND EXISTS (
           SELECT 1 FROM ai.document_roles r WHERE r.document_id = d.id AND r.role = %(role)s)))
"""
