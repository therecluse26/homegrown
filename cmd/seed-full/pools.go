package main

// ─── Name pools ───────────────────────────────────────────────────────────────

var lastNames = []string{
	"Anderson", "Baker", "Barnes", "Bell", "Bennett", "Brooks", "Brown", "Butler",
	"Campbell", "Carter", "Chen", "Clark", "Collins", "Cook", "Cooper", "Cox",
	"Cruz", "Davis", "Diaz", "Edwards", "Evans", "Fisher", "Flores", "Foster",
	"Garcia", "Gonzalez", "Gray", "Green", "Griffith", "Hall", "Hamilton", "Harris",
	"Harrison", "Hayes", "Henderson", "Hernandez", "Hill", "Holland", "Holmes",
	"Howard", "Hughes", "Hunt", "Jackson", "James", "Jenkins", "Johnson", "Jones",
	"Jordan", "Kelly", "Kennedy", "Kim", "King", "Knight", "Lee", "Lewis", "Li",
	"Long", "Lopez", "Martin", "Martinez", "Mason", "Matthews", "Miller", "Mitchell",
	"Moore", "Morgan", "Morris", "Murphy", "Murray", "Nelson", "Nguyen", "Olson",
	"Owens", "Palmer", "Parker", "Patel", "Patterson", "Perez", "Perry", "Peterson",
	"Phillips", "Porter", "Powell", "Price", "Quinn", "Ramirez", "Reed", "Reid",
	"Reyes", "Reynolds", "Richardson", "Rivera", "Roberts", "Robinson", "Rodriguez",
	"Rogers", "Ross", "Russell", "Ryan", "Sanders", "Scott", "Shaw", "Simmons",
	"Singh", "Smith", "Spencer", "Stewart", "Stone", "Sullivan", "Taylor", "Thomas",
	"Thompson", "Torres", "Turner", "Walker", "Wallace", "Walsh", "Wang", "Ward",
	"Washington", "Watson", "Webb", "Wells", "West", "White", "Williams", "Wilson",
	"Wood", "Wright", "Yang", "Young", "Abbott", "Aguilar", "Arnold", "Austin",
	"Bailey", "Baldwin", "Barker", "Bates", "Beck", "Bishop", "Blake", "Bowman",
	"Bradley", "Brennan", "Bryant", "Burke", "Burns", "Caldwell", "Carpenter",
	"Chambers", "Chapman", "Christensen", "Clayton", "Coleman", "Crawford", "Curtis",
	"Daniels", "Dean", "Douglas", "Duncan", "Dunn", "Elliott", "Ellis", "Ferguson",
	"Fleming", "Fletcher", "Ford", "Fox", "Franklin", "Freeman", "Garrett", "George",
	"Gibson", "Gilbert", "Gordon", "Graham", "Grant", "Greene", "Hale", "Hampton",
	"Hansen", "Hardy", "Harper", "Hawkins", "Haynes", "Hicks", "Hoffman", "Holland",
	"Hopkins", "Howell", "Hudson", "Ingram", "Jensen", "Johnston", "Keller", "Lamb",
	"Lane", "Lawson", "Leonard", "Lindsey", "Logan", "Lucas", "Lynch", "Marsh",
	"May", "McCarthy", "McDonald", "McKinney", "Mendez", "Meyer", "Miles", "Montgomery",
	"Moss", "Neal", "Newman", "Norman", "Obrien", "Oliver", "Ortiz", "Osborne",
	"Owen", "Page", "Parsons", "Paul", "Pearson", "Pena", "Peters", "Pierce",
}

var femaleNames = []string{
	"Abigail", "Addison", "Adelaide", "Adeline", "Alice", "Amelia", "Anna", "Annabelle",
	"Aria", "Audrey", "Aurora", "Autumn", "Ava", "Beatrice", "Brooklyn", "Camille",
	"Caroline", "Charlotte", "Chloe", "Clara", "Cora", "Dahlia", "Daisy", "Delilah",
	"Eleanor", "Elena", "Eliana", "Elizabeth", "Ella", "Emilia", "Emily", "Emma",
	"Evangeline", "Evelyn", "Felicity", "Fiona", "Florence", "Genevieve", "Georgia",
	"Grace", "Hadley", "Hannah", "Harper", "Hazel", "Iris", "Isabella", "Ivy",
	"Josephine", "Julia", "Juliana", "June", "Katherine", "Lila", "Lillian", "Lily",
	"Lucy", "Luna", "Lydia", "Madeline", "Mae", "Margaret", "Margot", "Maria",
	"Maya", "Mila", "Miriam", "Naomi", "Natalie", "Nora", "Olivia", "Penelope",
	"Phoebe", "Piper", "Quinn", "Rachel", "Rebecca", "Rosalie", "Rose", "Ruby",
	"Ruth", "Sadie", "Savannah", "Scarlett", "Sienna", "Sophia", "Stella", "Summer",
	"Thea", "Theodora", "Valentina", "Victoria", "Violet", "Vivian", "Willa", "Willow",
	"Wren", "Zara", "Zoey", "Lucia", "Esther", "Lyra", "Maeve", "Celeste",
}

var maleNames = []string{
	"Adrian", "Aiden", "Alexander", "Andrew", "Asher", "Atlas", "August", "Austin",
	"Benjamin", "Bennett", "Brooks", "Caleb", "Carter", "Charles", "Christian",
	"Christopher", "Cole", "Daniel", "David", "Declan", "Dominic", "Elias", "Elijah",
	"Emmett", "Ethan", "Everett", "Ezra", "Felix", "Finn", "Gabriel", "George",
	"Graham", "Grant", "Grayson", "Harrison", "Henry", "Hudson", "Hugh", "Ian",
	"Isaac", "Jack", "Jackson", "Jacob", "James", "Jasper", "John", "Jonah",
	"Joseph", "Joshua", "Julian", "Kai", "Knox", "Leo", "Levi", "Liam", "Lincoln",
	"Logan", "Lucas", "Luke", "Mason", "Matthew", "Micah", "Miles", "Nathaniel",
	"Nicholas", "Noah", "Nolan", "Oliver", "Oscar", "Owen", "Patrick", "Paul",
	"Peter", "Philip", "Rhys", "Roman", "Rowan", "Samuel", "Sebastian", "Silas",
	"Simon", "Sullivan", "Tate", "Theodore", "Thomas", "Timothy", "Tobias", "Wesley",
	"William", "Wyatt", "Xavier", "Zachary", "Beckett", "Calvin", "Cyrus", "Dean",
	"Edmund", "Elliott", "Frederick", "Griffin", "Harvey", "Holden", "Irving",
}

// ─── Geography ────────────────────────────────────────────────────────────────

var stateWeights = []weightedOption{
	{"TX", 12}, {"CA", 10}, {"FL", 8}, {"NC", 7}, {"VA", 6},
	{"OH", 5}, {"PA", 5}, {"GA", 5}, {"NY", 4}, {"IL", 4},
	{"TN", 4}, {"SC", 3}, {"AL", 3}, {"IN", 3}, {"MO", 3},
	{"WI", 2}, {"MI", 2}, {"MN", 2}, {"CO", 2}, {"OR", 2},
	{"WA", 2}, {"MD", 2}, {"LA", 1}, {"KY", 1}, {"AR", 1},
	{"MS", 1}, {"OK", 1}, {"KS", 1}, {"IA", 1}, {"NE", 1},
	{"NM", 1}, {"ID", 1}, {"UT", 1}, {"AZ", 1}, {"NV", 1},
	{"ME", 0.5}, {"NH", 0.5}, {"VT", 0.5}, {"CT", 0.5}, {"NJ", 0.5},
	{"WV", 0.5}, {"MT", 0.5}, {"ND", 0.5}, {"SD", 0.5}, {"WY", 0.5},
	{"AK", 0.3}, {"HI", 0.3}, {"DE", 0.3}, {"RI", 0.3}, {"DC", 0.2},
}

// stateNames maps abbreviation → full name for comply_state_configs.
var stateNames = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas",
	"CA": "California", "CO": "Colorado", "CT": "Connecticut", "DE": "Delaware",
	"DC": "District of Columbia", "FL": "Florida", "GA": "Georgia", "HI": "Hawaii",
	"ID": "Idaho", "IL": "Illinois", "IN": "Indiana", "IA": "Iowa",
	"KS": "Kansas", "KY": "Kentucky", "LA": "Louisiana", "ME": "Maine",
	"MD": "Maryland", "MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota",
	"MS": "Mississippi", "MO": "Missouri", "MT": "Montana", "NE": "Nebraska",
	"NV": "Nevada", "NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico",
	"NY": "New York", "NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio",
	"OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island",
	"SC": "South Carolina", "SD": "South Dakota", "TN": "Tennessee", "TX": "Texas",
	"UT": "Utah", "VT": "Vermont", "VA": "Virginia", "WA": "Washington",
	"WV": "West Virginia", "WI": "Wisconsin", "WY": "Wyoming",
}

var stateRegions = map[string]string{
	"TX": "Austin, TX", "CA": "Sacramento, CA", "FL": "Orlando, FL",
	"NC": "Raleigh, NC", "VA": "Richmond, VA", "OH": "Columbus, OH",
	"PA": "Philadelphia, PA", "GA": "Atlanta, GA", "NY": "Albany, NY",
	"IL": "Chicago, IL", "TN": "Nashville, TN", "SC": "Charleston, SC",
	"AL": "Birmingham, AL", "IN": "Indianapolis, IN", "MO": "St. Louis, MO",
	"WI": "Madison, WI", "MI": "Grand Rapids, MI", "MN": "Minneapolis, MN",
	"CO": "Denver, CO", "OR": "Portland, OR", "WA": "Seattle, WA",
	"MD": "Baltimore, MD", "LA": "Baton Rouge, LA", "KY": "Louisville, KY",
}

// ─── Methodology ──────────────────────────────────────────────────────────────

var methodologyWeights = []weightedOption{
	{"charlotte-mason", 25}, {"classical", 20}, {"traditional", 15},
	{"montessori", 15}, {"waldorf", 10}, {"unschooling", 15},
}

var methodologyDisplayNames = map[string]string{
	"charlotte-mason": "Charlotte Mason", "classical": "Classical",
	"traditional": "Traditional", "montessori": "Montessori",
	"waldorf": "Waldorf", "unschooling": "Unschooling",
}

// ─── Subject tags ─────────────────────────────────────────────────────────────

var subjectTags = []string{
	"mathematics", "reading", "language_arts", "writing", "science", "history",
	"geography", "nature_study", "art", "music", "latin", "french", "spanish",
	"physical_education", "life_skills", "bible_study", "logic", "rhetoric",
	"astronomy", "botany", "zoology", "chemistry", "physics", "civics",
	"economics", "philosophy", "poetry", "drama", "technology", "handicrafts",
}

// ─── Activity titles and descriptions ─────────────────────────────────────────

type activityTemplate struct {
	Title       string
	Description string
	Subjects    []string
	Duration    int // minutes
}

var activityTemplates = []activityTemplate{
	{"Nature Walk & Journal", "Outdoor observation walk followed by nature journaling with sketches and written narrations.", []string{"nature_study", "science", "art"}, 60},
	{"Math Games & Manipulatives", "Hands-on math using base-ten blocks, fraction tiles, and card games.", []string{"mathematics"}, 45},
	{"Read Aloud Session", "Family read aloud with discussion and narration.", []string{"reading", "language_arts"}, 30},
	{"Copywork Practice", "Careful handwriting practice copying passages from living books.", []string{"writing", "language_arts"}, 20},
	{"Dictation Exercise", "Listening dictation from prepared passage with self-correction.", []string{"writing", "language_arts"}, 25},
	{"Science Experiment", "Hands-on science investigation with hypothesis, procedure, and results.", []string{"science"}, 45},
	{"History Reading & Narration", "Living history book reading followed by oral or written narration.", []string{"history", "reading"}, 40},
	{"Timeline Work", "Adding entries to the personal history timeline with illustrations.", []string{"history", "art"}, 30},
	{"Latin Vocabulary Drill", "Review and practice of Latin vocabulary using flashcards and games.", []string{"latin"}, 20},
	{"Picture Study", "Careful observation of a master artwork followed by description from memory.", []string{"art"}, 25},
	{"Composer Study", "Listening to a featured composer's works with guided discussion.", []string{"music"}, 25},
	{"Poetry Recitation", "Memorization and recitation of poetry selections.", []string{"poetry", "language_arts"}, 15},
	{"Geography Map Work", "Map drawing, labeling, and geographic feature identification.", []string{"geography"}, 30},
	{"Watercolor Painting", "Guided watercolor technique practice with nature subjects.", []string{"art"}, 45},
	{"Hymn Study & Singing", "Learning and singing hymns with music theory discussion.", []string{"music"}, 20},
	{"French Conversation", "Conversational French practice with vocabulary building.", []string{"french"}, 25},
	{"Spanish Vocabulary", "Spanish word study with cultural context.", []string{"spanish"}, 25},
	{"Outdoor Physical Education", "Structured outdoor play, relay races, and fitness activities.", []string{"physical_education"}, 45},
	{"Logic Puzzles", "Working through age-appropriate logic problems and puzzles.", []string{"logic", "mathematics"}, 30},
	{"Creative Writing Workshop", "Free writing or prompted creative writing exercises.", []string{"writing", "language_arts"}, 35},
	{"Bird Watching Expedition", "Field identification of local bird species with field guide.", []string{"nature_study", "science"}, 60},
	{"Baking & Measurements", "Practical math through baking recipes and measurement conversions.", []string{"life_skills", "mathematics"}, 60},
	{"Astronomy Observation", "Stargazing and constellation identification.", []string{"astronomy", "science"}, 45},
	{"Botany Sketching", "Detailed botanical illustration of local plant specimens.", []string{"botany", "art", "science"}, 40},
	{"Handicraft: Knitting", "Learning or practicing knitting patterns and techniques.", []string{"handicrafts"}, 30},
	{"Drama & Shakespeare", "Reading and acting out scenes from Shakespeare adaptations.", []string{"drama", "language_arts"}, 40},
	{"Bible Study & Discussion", "Scripture reading with family discussion and journaling.", []string{"bible_study"}, 25},
	{"Swimming Lessons", "Structured swimming practice and water safety.", []string{"physical_education"}, 45},
	{"Civics & Government", "Learning about local and national government structures.", []string{"civics", "history"}, 30},
	{"Chemistry Basics", "Simple chemistry experiments exploring reactions and states of matter.", []string{"chemistry", "science"}, 40},
	{"Math Drills & Review", "Timed math fact practice for fluency building.", []string{"mathematics"}, 15},
	{"Independent Reading", "Self-selected book reading with reading log entry.", []string{"reading"}, 30},
	{"Gardening & Plant Care", "Hands-on garden work with plant identification and care.", []string{"nature_study", "life_skills"}, 45},
	{"Cooking Class", "Following a recipe with discussion of nutrition and food science.", []string{"life_skills", "science"}, 60},
	{"World Cultures Study", "Exploring a different culture through food, music, and stories.", []string{"geography", "history"}, 45},
	{"Phonics & Reading Practice", "Systematic phonics instruction and decodable reading.", []string{"reading", "language_arts"}, 20},
	{"Multiplication Tables", "Structured practice of multiplication facts.", []string{"mathematics"}, 15},
	{"Nature Journal Sketching", "Detailed outdoor sketching with written observations.", []string{"nature_study", "art"}, 40},
	{"Greek Mythology Reading", "Reading and discussing Greek myths with comprehension questions.", []string{"reading", "history"}, 35},
	{"Physics Fun", "Simple physics experiments exploring gravity, motion, and energy.", []string{"physics", "science"}, 40},
	{"Typing Practice", "Keyboarding skills development using typing software.", []string{"technology"}, 15},
	{"Wood Carving Basics", "Introduction to safe wood carving techniques.", []string{"handicrafts", "art"}, 45},
	{"Rhetoric Practice", "Persuasive speaking and argument construction exercises.", []string{"rhetoric", "language_arts"}, 30},
	{"Folk Song Study", "Learning traditional folk songs from various cultures.", []string{"music", "history"}, 20},
	{"Philosophy for Children", "Age-appropriate philosophical discussions using picture books.", []string{"philosophy", "reading"}, 30},
	{"Zoology Observation", "Animal behavior observation and classification activities.", []string{"zoology", "science"}, 35},
	{"Economics & Money", "Lessons on saving, spending, and basic economic concepts.", []string{"economics", "mathematics"}, 30},
	{"Pottery & Clay Work", "Hand-building pottery techniques and creative sculpting.", []string{"art", "handicrafts"}, 45},
	{"Weather Station", "Recording daily weather observations and learning meteorology basics.", []string{"science", "mathematics"}, 15},
	{"Sign Language Basics", "Learning American Sign Language vocabulary and phrases.", []string{"language_arts"}, 20},
}

// ─── Book catalog ─────────────────────────────────────────────────────────────

type bookTemplate struct {
	Title     string
	Author    string
	Subjects  []string
	PageCount int
}

var bookCatalog = []bookTemplate{
	{"Charlotte's Web", "E.B. White", []string{"reading", "language_arts"}, 184},
	{"The Lion, the Witch and the Wardrobe", "C.S. Lewis", []string{"reading"}, 208},
	{"Little House on the Prairie", "Laura Ingalls Wilder", []string{"reading", "history"}, 335},
	{"The Secret Garden", "Frances Hodgson Burnett", []string{"reading", "nature_study"}, 331},
	{"Treasure Island", "Robert Louis Stevenson", []string{"reading"}, 292},
	{"The Wind in the Willows", "Kenneth Grahame", []string{"reading", "nature_study"}, 258},
	{"Anne of Green Gables", "L.M. Montgomery", []string{"reading"}, 320},
	{"Black Beauty", "Anna Sewell", []string{"reading", "nature_study"}, 255},
	{"Heidi", "Johanna Spyri", []string{"reading", "geography"}, 288},
	{"The Hobbit", "J.R.R. Tolkien", []string{"reading"}, 310},
	{"Alice's Adventures in Wonderland", "Lewis Carroll", []string{"reading", "logic"}, 272},
	{"Peter Pan", "J.M. Barrie", []string{"reading", "drama"}, 248},
	{"The Jungle Book", "Rudyard Kipling", []string{"reading", "nature_study"}, 277},
	{"Robinson Crusoe", "Daniel Defoe", []string{"reading", "geography"}, 320},
	{"Swiss Family Robinson", "Johann David Wyss", []string{"reading", "science"}, 352},
	{"Pilgrim's Progress", "John Bunyan", []string{"reading", "bible_study"}, 336},
	{"A Bear Called Paddington", "Michael Bond", []string{"reading"}, 160},
	{"The Railway Children", "E. Nesbit", []string{"reading", "history"}, 224},
	{"The Story of the World Vol 1", "Susan Wise Bauer", []string{"history"}, 400},
	{"The Story of the World Vol 2", "Susan Wise Bauer", []string{"history"}, 416},
	{"The Story of the World Vol 3", "Susan Wise Bauer", []string{"history"}, 432},
	{"The Story of the World Vol 4", "Susan Wise Bauer", []string{"history"}, 448},
	{"D'Aulaires' Book of Greek Myths", "Ingri d'Aulaire", []string{"reading", "history"}, 192},
	{"D'Aulaires' Book of Norse Myths", "Ingri d'Aulaire", []string{"reading", "history"}, 160},
	{"Understood Betsy", "Dorothy Canfield Fisher", []string{"reading"}, 240},
	{"The Wheel on the School", "Meindert DeJong", []string{"reading", "nature_study"}, 298},
	{"Carry On, Mr. Bowditch", "Jean Lee Latham", []string{"reading", "mathematics", "history"}, 251},
	{"The Door in the Wall", "Marguerite de Angeli", []string{"reading", "history"}, 121},
	{"Amos Fortune, Free Man", "Elizabeth Yates", []string{"reading", "history"}, 192},
	{"Johnny Tremain", "Esther Forbes", []string{"reading", "history"}, 322},
	{"Island of the Blue Dolphins", "Scott O'Dell", []string{"reading", "nature_study"}, 192},
	{"Number the Stars", "Lois Lowry", []string{"reading", "history"}, 160},
	{"The Bronze Bow", "Elizabeth George Speare", []string{"reading", "history"}, 272},
	{"Caddie Woodlawn", "Carol Ryrie Brink", []string{"reading", "history"}, 286},
	{"The Trumpet of the Swan", "E.B. White", []string{"reading", "nature_study"}, 252},
	{"Stuart Little", "E.B. White", []string{"reading"}, 131},
	{"James and the Giant Peach", "Roald Dahl", []string{"reading"}, 160},
	{"Charlie and the Chocolate Factory", "Roald Dahl", []string{"reading"}, 176},
	{"Matilda", "Roald Dahl", []string{"reading"}, 240},
	{"The BFG", "Roald Dahl", []string{"reading"}, 208},
	{"Mrs. Frisby and the Rats of NIMH", "Robert C. O'Brien", []string{"reading", "science"}, 233},
	{"My Side of the Mountain", "Jean Craighead George", []string{"reading", "nature_study"}, 177},
	{"Julie of the Wolves", "Jean Craighead George", []string{"reading", "nature_study"}, 170},
	{"Hatchet", "Gary Paulsen", []string{"reading", "nature_study"}, 195},
	{"The Phantom Tollbooth", "Norton Juster", []string{"reading", "mathematics", "logic"}, 255},
	{"A Wrinkle in Time", "Madeleine L'Engle", []string{"reading", "science"}, 256},
	{"The Borrowers", "Mary Norton", []string{"reading"}, 180},
	{"The Cricket in Times Square", "George Selden", []string{"reading"}, 151},
	{"Pippi Longstocking", "Astrid Lindgren", []string{"reading"}, 160},
	{"The Complete Tales of Winnie-the-Pooh", "A.A. Milne", []string{"reading"}, 352},
	{"The Hundred Dresses", "Eleanor Estes", []string{"reading"}, 80},
	{"Sarah, Plain and Tall", "Patricia MacLachlan", []string{"reading"}, 64},
	{"The Courage of Sarah Noble", "Alice Dalgliesh", []string{"reading", "history"}, 52},
	{"Misty of Chincoteague", "Marguerite Henry", []string{"reading", "nature_study"}, 176},
	{"Rascal", "Sterling North", []string{"reading", "nature_study"}, 189},
	{"Handbook of Nature Study", "Anna Botsford Comstock", []string{"nature_study", "science"}, 887},
	{"Math-U-See Alpha", "Steve Demme", []string{"mathematics"}, 300},
	{"Math-U-See Beta", "Steve Demme", []string{"mathematics"}, 320},
	{"Math-U-See Gamma", "Steve Demme", []string{"mathematics"}, 340},
	{"Saxon Math 5/4", "Stephen Hake", []string{"mathematics"}, 640},
	{"Saxon Math 6/5", "Stephen Hake", []string{"mathematics"}, 680},
	{"Life of Fred: Fractions", "Stanley Schmidt", []string{"mathematics"}, 176},
	{"Life of Fred: Decimals", "Stanley Schmidt", []string{"mathematics"}, 192},
	{"Apologia Exploring Creation: Botany", "Jeannie Fulbright", []string{"botany", "science"}, 284},
	{"Apologia Exploring Creation: Astronomy", "Jeannie Fulbright", []string{"astronomy", "science"}, 272},
	{"Apologia Exploring Creation: Zoology", "Jeannie Fulbright", []string{"zoology", "science"}, 296},
	{"First Language Lessons", "Jessie Wise", []string{"language_arts", "writing"}, 310},
	{"Writing With Ease", "Susan Wise Bauer", []string{"writing", "language_arts"}, 384},
	{"Prima Latina", "Leigh Lowe", []string{"latin"}, 180},
	{"Latina Christiana", "Cheryl Lowe", []string{"latin"}, 220},
	{"Henle Latin First Year", "Robert Henle", []string{"latin"}, 520},
	{"The Good and the Beautiful Language Arts", "Jenny Phillips", []string{"language_arts", "reading"}, 350},
	{"All About Reading Level 1", "Marie Rippel", []string{"reading", "language_arts"}, 380},
	{"All About Reading Level 2", "Marie Rippel", []string{"reading", "language_arts"}, 400},
	{"All About Spelling Level 1", "Marie Rippel", []string{"writing", "language_arts"}, 290},
	{"Beautiful Girlhood", "Mabel Hale", []string{"reading", "life_skills"}, 224},
	{"Dangerous Journey (Pilgrim's Progress)", "Oliver Hunkin", []string{"reading", "bible_study"}, 128},
	{"The Children's Homer", "Padraic Colum", []string{"reading", "history"}, 260},
	{"Tales from Shakespeare", "Charles Lamb", []string{"reading", "drama"}, 384},
	{"A Child's Garden of Verses", "Robert Louis Stevenson", []string{"poetry", "reading"}, 110},
	{"Favorite Poems Old and New", "Helen Ferris", []string{"poetry", "reading"}, 598},
	{"This Country of Ours", "H.E. Marshall", []string{"history", "reading"}, 544},
	{"Our Island Story", "H.E. Marshall", []string{"history", "reading"}, 528},
	{"A Child's History of the World", "V.M. Hillyer", []string{"history"}, 608},
	{"Paddle-to-the-Sea", "Holling C. Holling", []string{"geography", "reading"}, 64},
	{"Tree in the Trail", "Holling C. Holling", []string{"history", "geography"}, 64},
	{"Seabird", "Holling C. Holling", []string{"history", "geography"}, 64},
	{"Pagoo", "Holling C. Holling", []string{"nature_study", "science"}, 64},
	{"Burgess Bird Book for Children", "Thornton W. Burgess", []string{"nature_study", "science"}, 400},
	{"Burgess Animal Book for Children", "Thornton W. Burgess", []string{"nature_study", "science"}, 376},
	{"The Story of Mankind", "Hendrik Willem van Loon", []string{"history"}, 480},
	{"The Little Prince", "Antoine de Saint-Exupéry", []string{"reading", "philosophy"}, 96},
	{"The Giving Tree", "Shel Silverstein", []string{"reading"}, 64},
	{"Where the Wild Things Are", "Maurice Sendak", []string{"reading"}, 48},
	{"Goodnight Moon", "Margaret Wise Brown", []string{"reading"}, 32},
	{"The Very Hungry Caterpillar", "Eric Carle", []string{"reading", "science"}, 26},
	{"Green Eggs and Ham", "Dr. Seuss", []string{"reading"}, 62},
	{"The Cat in the Hat", "Dr. Seuss", []string{"reading"}, 61},
	{"Brown Bear, Brown Bear", "Bill Martin Jr.", []string{"reading"}, 28},
	{"Caps for Sale", "Esphyr Slobodkina", []string{"reading"}, 48},
	{"Mike Mulligan and His Steam Shovel", "Virginia Lee Burton", []string{"reading", "history"}, 48},
	{"Make Way for Ducklings", "Robert McCloskey", []string{"reading", "nature_study"}, 68},
	{"Blueberries for Sal", "Robert McCloskey", []string{"reading", "nature_study"}, 54},
	{"One Morning in Maine", "Robert McCloskey", []string{"reading"}, 64},
	{"The Snowy Day", "Ezra Jack Keats", []string{"reading"}, 32},
	{"Harold and the Purple Crayon", "Crockett Johnson", []string{"reading", "art"}, 64},
	{"Curious George", "H.A. Rey", []string{"reading"}, 64},
	{"Frog and Toad Are Friends", "Arnold Lobel", []string{"reading"}, 64},
	{"Owl at Home", "Arnold Lobel", []string{"reading", "nature_study"}, 66},
	{"Ambleside Online Year 1 Collection", "Various Authors", []string{"reading", "language_arts"}, 500},
	{"Ambleside Online Year 2 Collection", "Various Authors", []string{"reading", "language_arts"}, 550},
	{"Ambleside Online Year 3 Collection", "Various Authors", []string{"reading", "history"}, 600},
	{"Ambleside Online Year 4 Collection", "Various Authors", []string{"reading", "history"}, 650},
	{"Song School Latin", "Classical Academic Press", []string{"latin"}, 160},
	{"Classical Conversations Foundations Guide", "Leigh Bortins", []string{"history", "science", "mathematics"}, 300},
	{"The Well-Trained Mind", "Susan Wise Bauer", []string{"reading"}, 816},
	{"Teaching the Trivium", "Harvey & Laurie Bluedorn", []string{"reading"}, 640},
	{"A Thomas Jefferson Education", "Oliver DeMille", []string{"reading"}, 240},
	{"For the Children's Sake", "Susan Schaeffer Macaulay", []string{"reading"}, 192},
	{"A Charlotte Mason Companion", "Karen Andreola", []string{"reading"}, 320},
	{"The Read-Aloud Handbook", "Jim Trelease", []string{"reading", "language_arts"}, 400},
	{"Dumbing Us Down", "John Taylor Gatto", []string{"reading"}, 148},
	{"Free to Learn", "Peter Gray", []string{"reading"}, 288},
	{"Unschooling Rules", "Clark Aldrich", []string{"reading"}, 160},
	{"Project-Based Homeschooling", "Lori Pickert", []string{"reading"}, 242},
	{"Last Child in the Woods", "Richard Louv", []string{"nature_study", "reading"}, 400},
	{"How to Raise a Wild Child", "Scott Sampson", []string{"nature_study", "reading"}, 352},
}

// ─── Social post templates ────────────────────────────────────────────────────

var postTemplates = []string{
	"Had such a wonderful nature walk today! We spotted three different species of birds and the kids filled two pages of their nature journals. 🌿",
	"Just finished our read-aloud of Charlotte's Web. The kids' narrations were so thoughtful — they really connected with the themes of friendship and loyalty.",
	"Our Charlotte Mason mornings have been going so well lately. Short lessons, living books, and lots of time outdoors. The rhythm feels right.",
	"Any recommendations for a good math curriculum for a 3rd grader who loves hands-on learning? We've been using manipulatives but want something more structured.",
	"Today's science experiment was a hit! We made a baking soda volcano and discussed chemical reactions. Simple but so effective for young learners.",
	"We just finished our first term of the school year. Feeling grateful for this homeschool journey and how much the kids have grown.",
	"Took the kids to the natural history museum today. Sometimes the best lessons happen outside the home! They were fascinated by the fossil exhibit.",
	"Started our new composer study this week — Vivaldi's Four Seasons. The kids are already humming Spring around the house!",
	"Morning basket time has become everyone's favorite part of the day. Poetry, hymns, and our current read-aloud all before 9am.",
	"Our co-op field trip to the botanical garden was amazing! The kids learned so much about native plants and pollinators.",
	"Just wanted to share that my reluctant reader finally found a book series he loves. Sometimes it just takes the right book at the right time.",
	"We're trying something new this week — nature journaling at the park every afternoon. The sketching skills are improving fast!",
	"Does anyone have tips for teaching cursive to a left-handed child? We're starting copywork and I want to make sure her grip is comfortable.",
	"Our family just finished reading The Lion, the Witch and the Wardrobe aloud. What a wonderful discussion about courage and sacrifice!",
	"Homeschool group meetup today was exactly what we needed. The kids played for hours while the moms talked curriculum over coffee.",
	"Finally organized our homeschool room! New bookshelves, a dedicated art station, and a cozy reading nook. It makes such a difference.",
	"Week 12 of our history cycle and the timeline is really coming together. The kids love adding illustrations for each new period.",
	"Picture study today: we spent 15 minutes observing a Monet painting, then described it from memory. Such a simple but powerful practice.",
	"Our vegetable garden is the best science lab we have. Today we measured growth, discussed photosynthesis, and harvested our first tomatoes!",
	"Latin lessons are going better than expected! The kids actually enjoy the chanting and the connection to English vocabulary roots.",
	"Made homemade bread today as part of our life skills and math lesson. Measuring, fractions, and the patience of waiting for dough to rise.",
	"We've been doing Shakespeare this term and I'm amazed at how much the kids understand when we read it aloud together with actions.",
	"Rainy day = library day! Each kid picked five books and we're set for the week. Our librarian always has the best recommendations.",
	"Today my 8-year-old narrated the entire chapter of our history book back to me, complete with dramatic hand gestures. This method works!",
	"Handicraft hour today: the older kids are learning to knit while the littles practice simple weaving. Peaceful, productive time.",
	"Any families in the Austin area interested in starting a Charlotte Mason study group? We'd love to connect with like-minded families.",
	"Just discovered the Handbook of Nature Study and it's completely transformed our outdoor time. Every walk becomes a learning adventure.",
	"Spent the morning at the creek catching crawdads and identifying water insects. Science class doesn't get better than this!",
	"Our first poetry tea time was a hit! Fancy cups, scones, and each child recited their memorized poem. Such a sweet tradition.",
	"Milestone: my youngest just read his first chapter book independently! Weeks of phonics practice paying off. So proud!",
	"We're wrapping up our ancient Egypt unit with a feast — the kids helped research and prepare ancient-inspired recipes.",
	"Tried unschooling for a week and while I loved the freedom, I think we need just a bit more structure. Finding our balance.",
	"The kids built an entire Roman aqueduct model out of cardboard today. Cross-curricular learning at its finest — engineering + history!",
	"Music appreciation this week: Bach's Brandenburg Concertos. My 6-year-old is convinced Bach was the greatest composer ever.",
	"Started our astronomy unit with a night sky observation. Spotted Jupiter and three constellations. The kids want a telescope now!",
	"We're halfway through our school year and I'm so encouraged by the progress. Sometimes you can't see growth until you look back.",
	"Art class today was painting local wildflowers we collected on our morning walk. Botanical illustration is surprisingly calming.",
	"Book recommendation: just finished reading aloud 'Carry On, Mr. Bowditch' and it was incredible. Math + history + adventure!",
	"Our homeschool group put on a small play today. The kids memorized lines, made costumes, and performed for families. Theatre magic!",
	"Anyone else find that their kids learn best in the morning? We front-load our hardest subjects and save afternoons for play and projects.",
	"Today we mapped our neighborhood and calculated distances. Real-world math is so much more engaging than worksheets!",
	"Made it through multiplication tables! Used songs, games, and lots of repetition. It took three months but the facts are solid now.",
	"We visited a working farm today. The kids learned about animal husbandry, crop rotation, and where their food actually comes from.",
	"Just ordered our curriculum for next year and I'm excited! Going with a Charlotte Mason approach for the first time.",
	"Our family nature journal is filling up beautifully. Each week we add new observations, pressed flowers, and sketches.",
	"Watercolor Wednesday is our new favorite tradition. No rules, just paint and paper and whatever inspires us.",
	"Had a great chat with our local librarian about building a reading list for my advanced reader. She's such a wonderful resource.",
	"Week 1 of our new schedule is done! Early mornings for focused work, afternoons for nature and free play. So far so good.",
	"The kids earned their swimming badges today! Physical education doesn't have to mean team sports. Water safety is a life skill.",
	"We're studying the solar system this month. Built a scale model in the backyard — the kids were shocked by how far apart planets are!",
}

// ─── Comment templates ────────────────────────────────────────────────────────

var commentTemplates = []string{
	"We love this approach! Going to try it with our family too.",
	"What book is this from? I'd love to add it to our reading list.",
	"This is so inspiring! Thanks for sharing your experience.",
	"We had a similar experience — it really does make a difference when you find the right rhythm.",
	"Beautiful! Your nature journal entries are always so lovely.",
	"Thank you for the recommendation! Just ordered it.",
	"We've been doing this too and the kids absolutely love it!",
	"Such a great idea! I never thought of combining those subjects that way.",
	"How long did it take your kids to adjust to this schedule?",
	"This is exactly what I needed to hear today. Thank you for the encouragement!",
	"We use the same curriculum and love it! Great choice.",
	"Would you mind sharing more details about how you structure this?",
	"My kids would love this! Adding it to our plans for next week.",
	"I'm so glad someone else does this too! Sometimes I feel like the only one.",
	"What a wonderful milestone! Congratulations to your little reader!",
	"We tried this last year and it transformed our homeschool. Highly recommend!",
	"Love seeing how other families approach this. Every family is so unique!",
	"Pinning this for later! Such a helpful resource.",
	"Your family sounds amazing. The kids are so lucky to have this experience!",
	"We just started this method and I'm already seeing positive changes. Thanks for the inspiration!",
}

// ─── Bio templates ────────────────────────────────────────────────────────────

var bioTemplates = []string{
	"Homeschooling family of %d in %s. Passionate about living books and nature study.",
	"Classical homeschoolers in %s. Year %d of our homeschool journey. Love learning together!",
	"Unschooling family in %s. Following our children's interests and loving every minute.",
	"%s homeschool family. %d kids, countless books, endless adventures.",
	"Charlotte Mason inspired family in %s. Nature walks, living books, and short lessons.",
	"Waldorf-inspired homeschoolers in %s. Rhythm, nature, and creativity guide our days.",
	"Montessori at home in %s. Hands-on learning for our %d littles.",
	"Eclectic homeschoolers picking the best from every method. Based in %s.",
	"Traditional homeschool family in %s. Structured days, strong academics, happy kids.",
	"Homeschooling since %d. %d kids, %s approach, and a lot of coffee.",
	"Nature-loving homeschool family in %s. You'll find us at the creek or in the garden!",
	"Bookworm family homeschooling in %s. Our library card gets a serious workout.",
	"New to homeschooling! Just started our %s journey in %s. Learning as we go!",
	"Second-generation homeschooler now teaching my own %d kids in %s.",
	"Military family homeschooling on the move. Currently stationed near %s.",
}

// ─── Group templates ──────────────────────────────────────────────────────────

type groupTemplate struct {
	Name        string
	Description string
	Methodology string // empty = any
	JoinPolicy  string
}

var groupTemplates = []groupTemplate{
	{"Charlotte Mason Study Circle", "A community for families following the Charlotte Mason method. Share ideas, resources, and encouragement.", "charlotte-mason", "open"},
	{"Classical Conversations Connect", "For families in Classical Conversations or other classical homeschool programs. Discuss the trivium and share resources.", "classical", "open"},
	{"Nature Study Enthusiasts", "Love nature study? Share your nature journal pages, field trip ideas, and favorite nature resources.", "", "open"},
	{"Waldorf Homeschool Collective", "Rhythm, handwork, and nature-based learning. Connect with other Waldorf-inspired families.", "waldorf", "open"},
	{"Montessori at Home", "Practical ideas for bringing Montessori principles into your homeschool. All ages welcome.", "montessori", "open"},
	{"Unschooling Explorers", "Child-led learning, interest-based education, and the freedom to explore. No curriculum required!", "unschooling", "open"},
	{"Homeschool Moms Book Club", "Monthly book discussions for homeschool parents. Current and past reads, always welcoming new members.", "", "open"},
	{"Texas Homeschool Network", "Connect with fellow Texas homeschoolers. State law updates, co-ops, and local meetups.", "", "open"},
	{"Southeast Homeschool Hub", "For homeschool families in the Southeast US. Share events, resources, and encouragement.", "", "open"},
	{"Midwest Homeschool Connect", "Connecting homeschool families across the Midwest. Meetups, field trips, and more.", "", "open"},
	{"West Coast Homeschoolers", "Homeschool community for families on the West Coast. Nature, co-ops, and curriculum chat.", "", "open"},
	{"Northeast Homeschool Alliance", "New England and Mid-Atlantic homeschoolers united. Historic field trips and academic resources.", "", "open"},
	{"Homeschool High School Prep", "Preparing for high school at home. Transcripts, SAT prep, college planning, and more.", "", "request_to_join"},
	{"Special Needs Homeschooling", "Support and resources for families homeschooling children with special needs. All are welcome.", "", "open"},
	{"Homeschool STEM Club", "Science, technology, engineering, and math projects for homeschool families. Monthly challenges!", "traditional", "open"},
	{"Read-Aloud Recommendations", "Share and discover the best books for reading aloud with your family.", "charlotte-mason", "open"},
	{"Homeschool Art Studio", "Art projects, picture study, and creative inspiration for homeschool families.", "waldorf", "open"},
	{"Latin & Languages Study Group", "For families studying Latin, Greek, or modern languages at home.", "classical", "open"},
	{"Homeschool Co-op Organizers", "Tips and strategies for starting and running successful homeschool co-ops.", "", "request_to_join"},
	{"Faith-Based Homeschooling", "Christian homeschool families sharing faith-centered curriculum and encouragement.", "traditional", "open"},
	{"Secular Homeschool Network", "Secular homeschooling resources, curriculum reviews, and community support.", "", "open"},
	{"Homeschool Dads", "A space for homeschool fathers. Tips, encouragement, and dad-led learning ideas.", "", "open"},
	{"Music & Movement", "Incorporating music education, movement, and performing arts into your homeschool.", "waldorf", "open"},
	{"Homeschool Gardeners", "Growing food, studying botany, and teaching life skills through gardening.", "montessori", "open"},
	{"History Buffs", "Deep dives into historical periods, living history books, and timeline projects.", "classical", "open"},
	{"Math Without Tears", "Making math enjoyable for reluctant learners. Games, manipulatives, and real-world applications.", "montessori", "open"},
	{"New Homeschoolers Welcome", "Just starting out? Ask questions, get advice, and find your footing in the homeschool world.", "", "open"},
	{"Homeschool Teens", "Activities, socialization, and community for homeschooled teenagers.", "", "open"},
	{"Preschool at Home", "Early childhood activities, play-based learning, and pre-reading skills for the littlest learners.", "montessori", "open"},
	{"Charlotte Mason Nature Study Group", "Weekly nature walk challenges, nature journal sharing, and seasonal nature study guides.", "charlotte-mason", "open"},
	{"Classical Education Discussion", "In-depth discussions about the trivium, great books, and classical pedagogy.", "classical", "request_to_join"},
	{"Homeschool Photography Club", "Capture your homeschool life! Photo challenges, tips, and sharing our favorite moments.", "", "open"},
	{"Creative Writing Corner", "For young writers and their parents. Writing prompts, workshops, and celebration of stories.", "", "open"},
	{"Field Trip Finders", "Discover and share amazing field trip destinations. Reviews, tips, and group outings.", "", "open"},
	{"Curriculum Swap & Share", "Buy, sell, and trade homeschool curriculum. One family's finished books are another's treasure!", "", "open"},
}

// ─── Event templates ──────────────────────────────────────────────────────────

type eventTemplate struct {
	Title       string
	Description string
	Category    string // methodology-specific or general
	IsVirtual   bool
	Capacity    int
}

var eventTemplates = []eventTemplate{
	{"Spring Nature Walk & Sketch", "Join us for a guided nature walk with time for nature journaling. Bring sketchbooks and colored pencils!", "", false, 25},
	{"Homeschool Park Day", "Weekly park meetup for homeschool families. Kids play while parents connect!", "", false, 50},
	{"Science Fair Showcase", "Present your science projects and experiments! Ribbons and certificates for all participants.", "", false, 40},
	{"Living History Day", "Come dressed as your favorite historical figure and share what you've learned!", "", false, 30},
	{"Poetry Tea Time", "Fancy tea, treats, and poetry recitation. Memorize a poem to share or just come to listen.", "", false, 20},
	{"Art Museum Field Trip", "Guided tour of the art museum with a picture study focus. All ages welcome.", "", false, 35},
	{"Homeschool Spelling Bee", "Friendly spelling competition with age-appropriate word lists. Prizes for all!", "", false, 30},
	{"Outdoor Games & PE Day", "Organized outdoor games, relay races, and team sports. Get active together!", "", false, 40},
	{"Book Swap Meet", "Bring books you've finished, swap for new treasures! Great way to refresh your library.", "", false, 30},
	{"Math Game Night", "Board games and card games that build math skills. Family-friendly fun!", "", false, 25},
	{"Nature Journaling Workshop", "Learn techniques for botanical illustration and nature observation recording.", "", false, 20},
	{"Homeschool Geography Bee", "Test your world knowledge! Maps, capitals, and geographic features challenge.", "", false, 30},
	{"Music Recital", "Share your musical talents! All instruments and skill levels welcome.", "", false, 35},
	{"Virtual Author Visit", "Live Q&A with a children's book author. Pre-read the featured book to participate!", "", true, 100},
	{"Co-op Registration Day", "Sign up for next semester's co-op classes. Browse offerings and meet teachers.", "", false, 60},
	{"Shakespeare in the Park", "Act out scenes from A Midsummer Night's Dream. Costumes encouraged!", "", false, 25},
	{"Stargazing Night", "Evening astronomy event with telescopes. Learn constellations and spot planets!", "", false, 30},
	{"Homeschool Talent Show", "Showcase your hidden talents — singing, magic, comedy, dance, anything goes!", "", false, 50},
	{"Farm Visit & Animal Care", "Visit a working farm to learn about animal husbandry and sustainable agriculture.", "", false, 25},
	{"Coding Workshop for Kids", "Introduction to programming using Scratch. Laptops provided.", "", false, 15},
	{"Virtual Book Club Meeting", "Monthly discussion of our current read-aloud selection. All ages participate!", "", true, 50},
	{"Pottery & Clay Workshop", "Hands-on pottery class. Make a pinch pot, coil pot, or free-form sculpture.", "", false, 15},
	{"Hiking & Nature Identification", "Moderate family hike with plant and animal identification along the trail.", "", false, 20},
	{"Homeschool Prom", "Formal event for homeschool teens. Dancing, photos, and making memories!", "", false, 80},
	{"End of Year Celebration", "Celebrate the school year! Awards, slideshow, and potluck dinner.", "", false, 100},
	{"Writing Workshop", "Creative writing workshop led by a published author. Bring notebooks and imagination!", "", false, 20},
	{"Lego Engineering Challenge", "Build the tallest tower, strongest bridge, or most creative vehicle. Legos provided!", "", false, 20},
	{"Spanish Immersion Playdate", "Play games and do activities entirely in Spanish. All levels welcome!", "", false, 15},
	{"Museum of Natural History Trip", "Docent-led tour focusing on geology and paleontology exhibits.", "", false, 35},
	{"Homeschool Swim Day", "Reserved pool time for homeschool families. Free swim plus organized water games.", "", false, 40},
}

// ─── Marketplace listing templates ────────────────────────────────────────────

type listingTemplate struct {
	Title       string
	Description string
	PriceCents  int
	ContentType string
	Subjects    []string
	GradeMin    int
	GradeMax    int
}

var listingTemplates = []listingTemplate{
	{"Nature Study Weekly Plans — Year 1", "52 weeks of guided nature study lessons with journaling prompts, identification guides, and seasonal activities.", 2999, "curriculum", []string{"nature_study", "science"}, 1, 6},
	{"Charlotte Mason Math Games Bundle", "40 printable math games aligned with CM philosophy. Hands-on, concrete-to-abstract progression.", 1499, "printable", []string{"mathematics"}, 1, 4},
	{"Medieval History Unit Study", "Complete 8-week unit study covering medieval Europe with living books list, activities, and assessments.", 1999, "unit_study", []string{"history"}, 3, 6},
	{"Copywork & Dictation Passages — Classic Literature", "100 carefully selected passages for copywork and dictation from timeless children's literature.", 899, "printable", []string{"writing", "language_arts"}, 1, 6},
	{"Watercolor Nature Journal Kit", "Video tutorials and printable templates for nature journal watercolor illustrations.", 2499, "curriculum", []string{"art", "nature_study"}, 1, 8},
	{"Latin Roots Vocabulary Cards", "200 Latin root word cards with definitions, examples, and memory aids.", 1299, "printable", []string{"latin", "language_arts"}, 3, 8},
	{"Astronomy for Young Explorers", "12-month astronomy curriculum with observation logs, constellation maps, and planet studies.", 3499, "curriculum", []string{"astronomy", "science"}, 2, 6},
	{"Poetry Memorization Tracker", "Beautiful printable tracker with 52 poems for the year, organized by difficulty and season.", 699, "printable", []string{"poetry", "language_arts"}, 1, 8},
	{"Hands-On Science Experiments — 100 Pack", "Step-by-step instructions for 100 science experiments using household materials.", 1999, "curriculum", []string{"science"}, 1, 6},
	{"Morning Basket Planning Kit", "Printable planners, rotation schedules, and resource lists for Charlotte Mason morning time.", 1199, "printable", []string{"reading", "language_arts"}, 0, 12},
	{"World Geography Lapbook", "Interactive lapbook project covering all 7 continents with maps, flags, and cultural activities.", 1499, "printable", []string{"geography"}, 2, 5},
	{"Classical Writing Curriculum — Year 1", "Complete rhetoric-stage writing program based on classical progymnasmata exercises.", 3999, "curriculum", []string{"writing", "rhetoric"}, 7, 9},
	{"Bible Study for Kids — Old Testament", "36-week Bible study with narration prompts, map work, and timeline activities.", 2499, "curriculum", []string{"bible_study", "history"}, 1, 6},
	{"Montessori Math Materials Guide", "Comprehensive guide to making and using Montessori math materials at home.", 1799, "curriculum", []string{"mathematics"}, 0, 3},
	{"French for Beginners — Family Course", "12-week conversational French course designed for families to learn together.", 2999, "curriculum", []string{"french"}, 1, 8},
	{"Composer Study Cards & Listening Guide", "30 composer study cards with biographical sketches and curated listening playlists.", 1099, "printable", []string{"music"}, 1, 8},
	{"Nature Walk Scavenger Hunts — Seasonal Set", "48 printable scavenger hunts organized by season and habitat type.", 899, "printable", []string{"nature_study", "science"}, 0, 6},
	{"Timeline Figures — Ancient History", "200+ printable timeline figures covering ancient civilizations with teacher notes.", 1999, "printable", []string{"history"}, 1, 6},
	{"Handicraft Skills Course — Knitting", "Video course teaching basic to intermediate knitting with 10 project patterns.", 2499, "curriculum", []string{"handicrafts"}, 3, 12},
	{"Reading Comprehension Through Narration", "Teacher guide with 50 narration exercises and comprehension strategies for living books.", 1599, "curriculum", []string{"reading", "language_arts"}, 1, 6},
	{"Waldorf Watercolor Painting Course", "Wet-on-wet watercolor technique lessons following Waldorf pedagogy. 24 guided paintings.", 2999, "curriculum", []string{"art"}, 0, 6},
	{"Life Skills Curriculum — Ages 6-12", "Weekly life skill lessons covering cooking, cleaning, money management, and more.", 1799, "curriculum", []string{"life_skills"}, 1, 6},
	{"Logic Puzzles & Critical Thinking — Level 1", "100 printable logic puzzles graded by difficulty for elementary students.", 999, "printable", []string{"logic", "mathematics"}, 2, 5},
	{"Spanish Through Stories", "16 illustrated stories in Spanish with vocabulary lists and comprehension activities.", 1999, "curriculum", []string{"spanish"}, 1, 4},
	{"Ancient Rome Unit Study", "6-week deep dive into Roman civilization. Includes recipes, crafts, and living books list.", 1799, "unit_study", []string{"history"}, 3, 6},
	{"Picture Study Prints — Renaissance Masters", "20 high-quality art prints with teacher guide for Charlotte Mason picture study.", 2499, "printable", []string{"art"}, 1, 8},
	{"Phonics Complete — Systematic Program", "36-week phonics program with decodable readers, games, and assessment tools.", 3499, "curriculum", []string{"reading", "language_arts"}, 0, 2},
	{"History Through Literature Book Lists", "Curated reading lists for 4 years of history, organized by time period and reading level.", 799, "book_list", []string{"history", "reading"}, 1, 8},
	{"STEM Challenge Cards — 50 Pack", "50 engineering design challenges using everyday materials. Perfect for co-ops!", 1299, "printable", []string{"science", "technology"}, 2, 8},
	{"Shakespeare for Young Readers", "Adapted scripts and study guides for 10 Shakespeare plays. Ages 8-14.", 1999, "curriculum", []string{"drama", "language_arts"}, 3, 8},
	{"Multiplication Mastery Game Pack", "Board games and card games specifically designed to build multiplication fluency.", 1099, "printable", []string{"mathematics"}, 2, 4},
	{"Botany Drawing Course", "12 video lessons teaching botanical illustration from basic shapes to detailed specimens.", 2499, "curriculum", []string{"botany", "art"}, 3, 12},
	{"Grammar Through Copywork", "180 copywork passages that systematically teach grammar concepts.", 999, "printable", []string{"writing", "language_arts"}, 2, 5},
	{"Ancient Egypt Unit Study", "Complete 6-week unit study with hands-on projects, recipes, and hieroglyphics activities.", 1799, "unit_study", []string{"history"}, 2, 5},
	{"Music Theory for Kids", "Interactive course teaching note reading, rhythm, and basic music theory through games.", 1999, "curriculum", []string{"music"}, 1, 6},
	{"Preschool Letter of the Week", "26-week printable curriculum with crafts, books, and activities for each letter.", 1499, "curriculum", []string{"reading", "language_arts"}, 0, 0},
	{"High School Chemistry Lab Manual", "At-home chemistry lab experiments with safety guidelines and detailed procedures.", 2999, "curriculum", []string{"chemistry", "science"}, 9, 12},
	{"World Religions Study Guide", "Comparative study of major world religions for upper elementary and middle school.", 1599, "curriculum", []string{"history", "philosophy"}, 5, 8},
	{"Creative Writing Prompts — 365 Days", "A year of daily creative writing prompts organized by theme and season.", 799, "printable", []string{"writing"}, 2, 8},
	{"Physical Education Game Cards", "52 outdoor PE games requiring minimal equipment. Great for co-ops and park days.", 899, "printable", []string{"physical_education"}, 0, 8},
}

// ─── Journal entry templates ──────────────────────────────────────────────────

var journalTemplates = []struct {
	Title   string
	Content string
	Type    string
}{
	{"Today's Nature Walk", "Today we went on a nature walk and I saw %s. The weather was %s and I drew a picture of %s in my journal.", "narration"},
	{"My Favorite Book Chapter", "The chapter we read today was about %s. I think the most interesting part was when %s. It reminded me of %s.", "narration"},
	{"Science Experiment Results", "We did an experiment with %s today. My hypothesis was that %s. The result was %s!", "narration"},
	{"What I Learned in History", "Today I learned about %s. I found it interesting that %s. I would like to learn more about %s.", "narration"},
	{"My Garden Observations", "The plants in our garden are %s. I measured the tallest one and it was %s inches. I noticed %s.", "narration"},
	{"Free Writing", "Today I want to write about my favorite hobby which is %s. I like it because %s. My goal is to %s.", "freeform"},
	{"My Drawing", "I drew a picture of %s today. I used %s for the colors. I'm proud of how the %s turned out.", "freeform"},
	{"Letter to a Friend", "Dear friend, today I want to tell you about %s. It was really %s. I wish you could have been there to see %s.", "freeform"},
	{"Weekend Reflection", "This weekend we %s. My favorite part was %s. Next weekend I hope we can %s.", "reflection"},
	{"Book Report", "The book I just finished was really %s. The main character had to overcome %s. I would recommend this book to %s.", "narration"},
}

// ─── Review templates ─────────────────────────────────────────────────────────

var reviewTemplates = []struct {
	Text   string
	Rating int
}{
	{"Absolutely love this curriculum! It's been transformative for our homeschool. Well-organized and easy to follow.", 5},
	{"Great resource. My kids enjoy the activities and I appreciate the thoughtful design. Highly recommend!", 5},
	{"Very good quality. The printables are beautiful and the content is solid. Worth every penny.", 5},
	{"Good resource but could use more variety in activities. Still, we've gotten a lot of use out of it.", 4},
	{"Solid curriculum. Not flashy but gets the job done. My kids are learning and that's what matters.", 4},
	{"Really well thought out. You can tell the creator is an experienced homeschool parent.", 5},
	{"Nice addition to our homeschool. The kids enjoy the hands-on elements. Would buy from this creator again.", 4},
	{"Decent quality. Some sections are stronger than others but overall a good purchase.", 3},
	{"Perfect for our Charlotte Mason approach! Integrates beautifully with our existing routine.", 5},
	{"We've been using this for two months and the progress is noticeable. Thank you for creating this!", 5},
	{"Good for the price. Not as comprehensive as I hoped but still useful. Would recommend for beginners.", 3},
	{"My daughter loves this! She asks to do these lessons every day. That's the best review I can give.", 5},
	{"Well-designed and age-appropriate. The scope and sequence are excellent.", 4},
	{"Exceeded my expectations! The quality of the content and the attention to detail are impressive.", 5},
	{"Helpful but I wish there were more teacher notes. Some lessons need more guidance for parents.", 3},
}

// ─── Publisher names ──────────────────────────────────────────────────────────

var publisherNames = []struct {
	Name string
	Slug string
	Desc string
}{
	{"Living Books Press", "living-books-press", "Publisher of Charlotte Mason-aligned curriculum and nature study resources."},
	{"Trivium Academy", "trivium-academy", "Classical education materials rooted in the trivium tradition."},
	{"Wildflower Learning", "wildflower-learning", "Nature-based curriculum for Waldorf and Charlotte Mason families."},
	{"Heritage Curriculum Co", "heritage-curriculum-co", "Faith-based homeschool curriculum with a focus on American heritage."},
	{"Discovery STEM Lab", "discovery-stem-lab", "Hands-on STEM resources designed for homeschool families."},
	{"Fireside Education", "fireside-education", "Cozy, literature-rich curriculum for the whole family."},
	{"Golden Ratio Press", "golden-ratio-press", "Mathematics curriculum that makes math beautiful and accessible."},
	{"Little Scholars", "little-scholars", "Preschool and early elementary resources for the home educator."},
}

// ─── Notification templates ───────────────────────────────────────────────────

type notifTemplate struct {
	Type     string
	Category string
	Title    string
	Body     string
}

var notifTemplates = []notifTemplate{
	{"friend_request_sent", "social", "Friend Request Sent", "You sent a friend request to a family."},
	{"friend_request_accepted", "social", "Friend Request Accepted", "Your friend request has been accepted."},
	{"message_received", "social", "New Message", "You have a new direct message."},
	{"event_cancelled", "social", "Event Cancelled", "An event you RSVP'd to has been cancelled."},
	{"co_parent_added", "social", "Co-Parent Added", "A co-parent has been added to your family."},
	{"methodology_changed", "system", "Methodology Updated", "Your family's methodology has been updated."},
	{"onboarding_completed", "system", "Welcome!", "You've completed onboarding — start exploring!"},
	{"activity_streak", "learning", "Learning Streak!", "Keep up the great work! You've been consistent this week."},
	{"milestone_achieved", "learning", "Milestone Achieved!", "A student reached a new learning milestone."},
	{"book_completed", "learning", "Book Completed!", "Congratulations on finishing another book!"},
	{"data_export_ready", "system", "Export Ready", "Your data export is ready for download."},
	{"purchase_completed", "marketplace", "Purchase Complete", "Your marketplace purchase has been confirmed."},
	{"purchase_refunded", "marketplace", "Refund Processed", "Your marketplace refund has been processed."},
	{"creator_onboarded", "marketplace", "Creator Welcome", "You're now set up as a marketplace creator!"},
	{"subscription_created", "system", "Subscription Active", "Your premium subscription is now active."},
	{"subscription_cancelled", "system", "Subscription Cancelled", "Your premium subscription has been cancelled."},
}

// ─── Schedule category templates ──────────────────────────────────────────────

var scheduleCategories = []string{
	"lesson", "reading", "activity", "assessment", "field_trip", "co_op", "break", "custom",
}

