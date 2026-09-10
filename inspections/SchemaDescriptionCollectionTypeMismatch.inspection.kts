import com.goide.psi.GoCompositeLit
import com.goide.psi.GoFile
import org.intellij.lang.annotations.Language
import java.util.regex.Pattern

/**
 * A collection attribute whose MarkdownDescription names a different collection kind. The wording
 * lands verbatim in the registry docs next to the type tfplugindocs derives from the schema, so the
 * two contradict each other on the published page: "(Attributes Set) List of parent building blocks".
 *
 * This is what commit de88f74 fixed by hand across four files. It happens when an attribute's type
 * changes and the description keeps the old wording.
 */

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

// java.util.regex, not kotlin.text.Regex: the script runs under its own REPL classloader, and
// touching a kotlin.text.MatchResult there throws a LinkageError, because the IDE's plugin
// classloader has already defined that interface. The JDK regex classes come from the bootstrap
// loader, so both sides agree on them. The failure is silent — the inspection compiles, registers,
// and then reports nothing while the exception goes to idea.log.
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
