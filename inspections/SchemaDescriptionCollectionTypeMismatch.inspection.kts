import com.goide.psi.GoCompositeLit
import com.goide.psi.GoFile
import org.intellij.lang.annotations.Language
import java.util.regex.Pattern

@Language("HTML")
val htmlDescription =
    """
    <html>
    <body>
        <p>The attribute's <code>MarkdownDescription</code> names a different collection kind than the
        attribute's type.</p>
        <p><code>tfplugindocs</code> prints the schema type and the description side by side, so the published
        page reads <i>"(Attributes Set) List of parent building blocks"</i>. Terraform users rely on that
        wording to know whether ordering matters.</p>
    </body>
    </html>
    """.trimIndent()

private val attributeType: Pattern = Pattern.compile("""schema\.(\w+?)(Nested)?(Attribute|Block)""")

// "map of" is left out on purpose: a Map attribute is often legitimately described as
// "map of X" while a Set of objects keyed by name is described as a "map" in prose.
private val wrongWordingFor: Map<String, Pattern> =
    mapOf(
        "Set" to Pattern.compile("""\blists? of\b""", Pattern.CASE_INSENSITIVE),
        "List" to Pattern.compile("""\bsets? of\b""", Pattern.CASE_INSENSITIVE),
    )

val schemaDescriptionCollectionTypeMismatch =
    localInspection { psiFile, inspection ->
        val goFile = psiFile as? GoFile ?: return@localInspection

        goFile.descendantsOfType<GoCompositeLit>().forEach { lit ->
            val declaredType = lit.typeReferenceExpression?.text.orEmpty()
            val typeMatcher = attributeType.matcher(declaredType)
            if (!typeMatcher.matches()) return@forEach
            val wrongWording = wrongWordingFor[typeMatcher.group(1)] ?: return@forEach

            val description =
                lit.literalValue
                    ?.elementList
                    .orEmpty()
                    .firstOrNull { it.key?.text == "MarkdownDescription" } ?: return@forEach
            val wordingMatcher = wrongWording.matcher(description.value?.text ?: return@forEach)
            if (!wordingMatcher.find()) return@forEach

            inspection.registerProblem(
                description,
                "This is a $declaredType but the description says \"${wordingMatcher.group()}\"; " +
                    "the registry page will print the schema type and this wording next to each other.",
            )
        }
    }

listOf(
    InspectionKts(
        id = "SchemaDescriptionCollectionTypeMismatch",
        localTool = schemaDescriptionCollectionTypeMismatch,
        name = "Schema description names a different collection kind than the attribute type",
        htmlDescription = htmlDescription,
        level = HighlightDisplayLevel.WARNING,
    ),
)
