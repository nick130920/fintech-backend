package currency

import (
	"fmt"
	"strings"
)

type Info struct {
	Code     string `json:"iso_code"`
	Numeric  string `json:"iso_numeric"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

var byCode map[string]Info

func init() {
	byCode = make(map[string]Info, len(all))
	for _, c := range all {
		byCode[c.Code] = c
	}
}

func IsValid(code string) bool {
	_, ok := byCode[strings.ToUpper(code)]
	return ok
}

func Get(code string) (Info, bool) {
	info, ok := byCode[strings.ToUpper(code)]
	return info, ok
}

func All() []Info {
	out := make([]Info, len(all))
	copy(out, all)
	return out
}

func Format(amount float64, code string) string {
	info, ok := byCode[strings.ToUpper(code)]
	if !ok {
		return fmt.Sprintf("%.2f %s", amount, code)
	}
	switch info.Decimals {
	case 0:
		return fmt.Sprintf("%s %.0f", info.Symbol, amount)
	case 3:
		return fmt.Sprintf("%s %.3f", info.Symbol, amount)
	default:
		return fmt.Sprintf("%s %.2f", info.Symbol, amount)
	}
}

const DefaultCurrency = "USD"

var all = []Info{
	{Code: "AED", Numeric: "784", Name: "UAE Dirham", Symbol: "د.إ", Decimals: 2},
	{Code: "AFN", Numeric: "971", Name: "Afghan Afghani", Symbol: "؋", Decimals: 2},
	{Code: "ALL", Numeric: "008", Name: "Albanian Lek", Symbol: "L", Decimals: 2},
	{Code: "AMD", Numeric: "051", Name: "Armenian Dram", Symbol: "֏", Decimals: 2},
	{Code: "ANG", Numeric: "532", Name: "Netherlands Antillean Guilder", Symbol: "ƒ", Decimals: 2},
	{Code: "AOA", Numeric: "973", Name: "Angolan Kwanza", Symbol: "Kz", Decimals: 2},
	{Code: "ARS", Numeric: "032", Name: "Argentine Peso", Symbol: "$", Decimals: 2},
	{Code: "AUD", Numeric: "036", Name: "Australian Dollar", Symbol: "A$", Decimals: 2},
	{Code: "AWG", Numeric: "533", Name: "Aruban Florin", Symbol: "ƒ", Decimals: 2},
	{Code: "AZN", Numeric: "944", Name: "Azerbaijani Manat", Symbol: "₼", Decimals: 2},
	{Code: "BAM", Numeric: "977", Name: "Bosnia-Herzegovina Convertible Mark", Symbol: "KM", Decimals: 2},
	{Code: "BBD", Numeric: "052", Name: "Barbadian Dollar", Symbol: "Bds$", Decimals: 2},
	{Code: "BDT", Numeric: "050", Name: "Bangladeshi Taka", Symbol: "৳", Decimals: 2},
	{Code: "BGN", Numeric: "975", Name: "Bulgarian Lev", Symbol: "лв.", Decimals: 2},
	{Code: "BHD", Numeric: "048", Name: "Bahraini Dinar", Symbol: ".د.ب", Decimals: 3},
	{Code: "BIF", Numeric: "108", Name: "Burundian Franc", Symbol: "FBu", Decimals: 0},
	{Code: "BMD", Numeric: "060", Name: "Bermudian Dollar", Symbol: "$", Decimals: 2},
	{Code: "BND", Numeric: "096", Name: "Brunei Dollar", Symbol: "B$", Decimals: 2},
	{Code: "BOB", Numeric: "068", Name: "Bolivian Boliviano", Symbol: "Bs.", Decimals: 2},
	{Code: "BRL", Numeric: "986", Name: "Brazilian Real", Symbol: "R$", Decimals: 2},
	{Code: "BSD", Numeric: "044", Name: "Bahamian Dollar", Symbol: "$", Decimals: 2},
	{Code: "BTN", Numeric: "064", Name: "Bhutanese Ngultrum", Symbol: "Nu.", Decimals: 2},
	{Code: "BWP", Numeric: "072", Name: "Botswanan Pula", Symbol: "P", Decimals: 2},
	{Code: "BYN", Numeric: "933", Name: "Belarusian Ruble", Symbol: "Br", Decimals: 2},
	{Code: "BZD", Numeric: "084", Name: "Belize Dollar", Symbol: "BZ$", Decimals: 2},
	{Code: "CAD", Numeric: "124", Name: "Canadian Dollar", Symbol: "C$", Decimals: 2},
	{Code: "CDF", Numeric: "976", Name: "Congolese Franc", Symbol: "FC", Decimals: 2},
	{Code: "CHF", Numeric: "756", Name: "Swiss Franc", Symbol: "CHF", Decimals: 2},
	{Code: "CLP", Numeric: "152", Name: "Chilean Peso", Symbol: "$", Decimals: 0},
	{Code: "CNY", Numeric: "156", Name: "Chinese Yuan", Symbol: "¥", Decimals: 2},
	{Code: "COP", Numeric: "170", Name: "Colombian Peso", Symbol: "$", Decimals: 0},
	{Code: "CRC", Numeric: "188", Name: "Costa Rican Colón", Symbol: "₡", Decimals: 2},
	{Code: "CUP", Numeric: "192", Name: "Cuban Peso", Symbol: "$", Decimals: 2},
	{Code: "CVE", Numeric: "132", Name: "Cape Verdean Escudo", Symbol: "$", Decimals: 2},
	{Code: "CZK", Numeric: "203", Name: "Czech Koruna", Symbol: "Kč", Decimals: 2},
	{Code: "DJF", Numeric: "262", Name: "Djiboutian Franc", Symbol: "Fdj", Decimals: 0},
	{Code: "DKK", Numeric: "208", Name: "Danish Krone", Symbol: "kr", Decimals: 2},
	{Code: "DOP", Numeric: "214", Name: "Dominican Peso", Symbol: "RD$", Decimals: 2},
	{Code: "DZD", Numeric: "012", Name: "Algerian Dinar", Symbol: "د.ج", Decimals: 2},
	{Code: "EGP", Numeric: "818", Name: "Egyptian Pound", Symbol: "E£", Decimals: 2},
	{Code: "ERN", Numeric: "232", Name: "Eritrean Nakfa", Symbol: "Nfk", Decimals: 2},
	{Code: "ETB", Numeric: "230", Name: "Ethiopian Birr", Symbol: "Br", Decimals: 2},
	{Code: "EUR", Numeric: "978", Name: "Euro", Symbol: "€", Decimals: 2},
	{Code: "FJD", Numeric: "242", Name: "Fijian Dollar", Symbol: "FJ$", Decimals: 2},
	{Code: "GBP", Numeric: "826", Name: "British Pound", Symbol: "£", Decimals: 2},
	{Code: "GEL", Numeric: "981", Name: "Georgian Lari", Symbol: "₾", Decimals: 2},
	{Code: "GHS", Numeric: "936", Name: "Ghanaian Cedi", Symbol: "GH₵", Decimals: 2},
	{Code: "GIP", Numeric: "292", Name: "Gibraltar Pound", Symbol: "£", Decimals: 2},
	{Code: "GMD", Numeric: "270", Name: "Gambian Dalasi", Symbol: "D", Decimals: 2},
	{Code: "GNF", Numeric: "324", Name: "Guinean Franc", Symbol: "FG", Decimals: 0},
	{Code: "GTQ", Numeric: "320", Name: "Guatemalan Quetzal", Symbol: "Q", Decimals: 2},
	{Code: "GYD", Numeric: "328", Name: "Guyanaese Dollar", Symbol: "GY$", Decimals: 2},
	{Code: "HKD", Numeric: "344", Name: "Hong Kong Dollar", Symbol: "HK$", Decimals: 2},
	{Code: "HNL", Numeric: "340", Name: "Honduran Lempira", Symbol: "L", Decimals: 2},
	{Code: "HRK", Numeric: "191", Name: "Croatian Kuna", Symbol: "kn", Decimals: 2},
	{Code: "HTG", Numeric: "332", Name: "Haitian Gourde", Symbol: "G", Decimals: 2},
	{Code: "HUF", Numeric: "348", Name: "Hungarian Forint", Symbol: "Ft", Decimals: 2},
	{Code: "IDR", Numeric: "360", Name: "Indonesian Rupiah", Symbol: "Rp", Decimals: 2},
	{Code: "ILS", Numeric: "376", Name: "Israeli New Shekel", Symbol: "₪", Decimals: 2},
	{Code: "INR", Numeric: "356", Name: "Indian Rupee", Symbol: "₹", Decimals: 2},
	{Code: "IQD", Numeric: "368", Name: "Iraqi Dinar", Symbol: "ع.د", Decimals: 3},
	{Code: "IRR", Numeric: "364", Name: "Iranian Rial", Symbol: "﷼", Decimals: 2},
	{Code: "ISK", Numeric: "352", Name: "Icelandic Króna", Symbol: "kr", Decimals: 0},
	{Code: "JMD", Numeric: "388", Name: "Jamaican Dollar", Symbol: "J$", Decimals: 2},
	{Code: "JOD", Numeric: "400", Name: "Jordanian Dinar", Symbol: "JD", Decimals: 3},
	{Code: "JPY", Numeric: "392", Name: "Japanese Yen", Symbol: "¥", Decimals: 0},
	{Code: "KES", Numeric: "404", Name: "Kenyan Shilling", Symbol: "KSh", Decimals: 2},
	{Code: "KGS", Numeric: "417", Name: "Kyrgystani Som", Symbol: "сом", Decimals: 2},
	{Code: "KHR", Numeric: "116", Name: "Cambodian Riel", Symbol: "៛", Decimals: 2},
	{Code: "KMF", Numeric: "174", Name: "Comorian Franc", Symbol: "CF", Decimals: 0},
	{Code: "KRW", Numeric: "410", Name: "South Korean Won", Symbol: "₩", Decimals: 0},
	{Code: "KWD", Numeric: "414", Name: "Kuwaiti Dinar", Symbol: "د.ك", Decimals: 3},
	{Code: "KYD", Numeric: "136", Name: "Cayman Islands Dollar", Symbol: "CI$", Decimals: 2},
	{Code: "KZT", Numeric: "398", Name: "Kazakhstani Tenge", Symbol: "₸", Decimals: 2},
	{Code: "LAK", Numeric: "418", Name: "Laotian Kip", Symbol: "₭", Decimals: 2},
	{Code: "LBP", Numeric: "422", Name: "Lebanese Pound", Symbol: "ل.ل", Decimals: 2},
	{Code: "LKR", Numeric: "144", Name: "Sri Lankan Rupee", Symbol: "Rs", Decimals: 2},
	{Code: "LRD", Numeric: "430", Name: "Liberian Dollar", Symbol: "L$", Decimals: 2},
	{Code: "LSL", Numeric: "426", Name: "Lesotho Loti", Symbol: "L", Decimals: 2},
	{Code: "MAD", Numeric: "504", Name: "Moroccan Dirham", Symbol: "MAD", Decimals: 2},
	{Code: "MDL", Numeric: "498", Name: "Moldovan Leu", Symbol: "L", Decimals: 2},
	{Code: "MGA", Numeric: "969", Name: "Malagasy Ariary", Symbol: "Ar", Decimals: 2},
	{Code: "MKD", Numeric: "807", Name: "Macedonian Denar", Symbol: "ден", Decimals: 2},
	{Code: "MMK", Numeric: "104", Name: "Myanmar Kyat", Symbol: "K", Decimals: 2},
	{Code: "MNT", Numeric: "496", Name: "Mongolian Tugrik", Symbol: "₮", Decimals: 2},
	{Code: "MOP", Numeric: "446", Name: "Macanese Pataca", Symbol: "MOP$", Decimals: 2},
	{Code: "MRU", Numeric: "929", Name: "Mauritanian Ouguiya", Symbol: "UM", Decimals: 2},
	{Code: "MUR", Numeric: "480", Name: "Mauritian Rupee", Symbol: "₨", Decimals: 2},
	{Code: "MVR", Numeric: "462", Name: "Maldivian Rufiyaa", Symbol: "Rf", Decimals: 2},
	{Code: "MWK", Numeric: "454", Name: "Malawian Kwacha", Symbol: "MK", Decimals: 2},
	{Code: "MXN", Numeric: "484", Name: "Mexican Peso", Symbol: "$", Decimals: 2},
	{Code: "MYR", Numeric: "458", Name: "Malaysian Ringgit", Symbol: "RM", Decimals: 2},
	{Code: "MZN", Numeric: "943", Name: "Mozambican Metical", Symbol: "MT", Decimals: 2},
	{Code: "NAD", Numeric: "516", Name: "Namibian Dollar", Symbol: "N$", Decimals: 2},
	{Code: "NGN", Numeric: "566", Name: "Nigerian Naira", Symbol: "₦", Decimals: 2},
	{Code: "NIO", Numeric: "558", Name: "Nicaraguan Córdoba", Symbol: "C$", Decimals: 2},
	{Code: "NOK", Numeric: "578", Name: "Norwegian Krone", Symbol: "kr", Decimals: 2},
	{Code: "NPR", Numeric: "524", Name: "Nepalese Rupee", Symbol: "₨", Decimals: 2},
	{Code: "NZD", Numeric: "554", Name: "New Zealand Dollar", Symbol: "NZ$", Decimals: 2},
	{Code: "OMR", Numeric: "512", Name: "Omani Rial", Symbol: "ر.ع.", Decimals: 3},
	{Code: "PAB", Numeric: "590", Name: "Panamanian Balboa", Symbol: "B/.", Decimals: 2},
	{Code: "PEN", Numeric: "604", Name: "Peruvian Sol", Symbol: "S/", Decimals: 2},
	{Code: "PGK", Numeric: "598", Name: "Papua New Guinean Kina", Symbol: "K", Decimals: 2},
	{Code: "PHP", Numeric: "608", Name: "Philippine Peso", Symbol: "₱", Decimals: 2},
	{Code: "PKR", Numeric: "586", Name: "Pakistani Rupee", Symbol: "₨", Decimals: 2},
	{Code: "PLN", Numeric: "985", Name: "Polish Zloty", Symbol: "zł", Decimals: 2},
	{Code: "PYG", Numeric: "600", Name: "Paraguayan Guarani", Symbol: "₲", Decimals: 0},
	{Code: "QAR", Numeric: "634", Name: "Qatari Rial", Symbol: "ر.ق", Decimals: 2},
	{Code: "RON", Numeric: "946", Name: "Romanian Leu", Symbol: "lei", Decimals: 2},
	{Code: "RSD", Numeric: "941", Name: "Serbian Dinar", Symbol: "din.", Decimals: 2},
	{Code: "RUB", Numeric: "643", Name: "Russian Ruble", Symbol: "₽", Decimals: 2},
	{Code: "RWF", Numeric: "646", Name: "Rwandan Franc", Symbol: "RF", Decimals: 0},
	{Code: "SAR", Numeric: "682", Name: "Saudi Riyal", Symbol: "ر.س", Decimals: 2},
	{Code: "SBD", Numeric: "090", Name: "Solomon Islands Dollar", Symbol: "SI$", Decimals: 2},
	{Code: "SCR", Numeric: "690", Name: "Seychellois Rupee", Symbol: "₨", Decimals: 2},
	{Code: "SDG", Numeric: "938", Name: "Sudanese Pound", Symbol: "ج.س.", Decimals: 2},
	{Code: "SEK", Numeric: "752", Name: "Swedish Krona", Symbol: "kr", Decimals: 2},
	{Code: "SGD", Numeric: "702", Name: "Singapore Dollar", Symbol: "S$", Decimals: 2},
	{Code: "SHP", Numeric: "654", Name: "Saint Helena Pound", Symbol: "£", Decimals: 2},
	{Code: "SLL", Numeric: "694", Name: "Sierra Leonean Leone", Symbol: "Le", Decimals: 2},
	{Code: "SOS", Numeric: "706", Name: "Somali Shilling", Symbol: "Sh", Decimals: 2},
	{Code: "SRD", Numeric: "968", Name: "Surinamese Dollar", Symbol: "$", Decimals: 2},
	{Code: "SSP", Numeric: "728", Name: "South Sudanese Pound", Symbol: "£", Decimals: 2},
	{Code: "STN", Numeric: "930", Name: "São Tomé and Príncipe Dobra", Symbol: "Db", Decimals: 2},
	{Code: "SYP", Numeric: "760", Name: "Syrian Pound", Symbol: "£S", Decimals: 2},
	{Code: "SZL", Numeric: "748", Name: "Swazi Lilangeni", Symbol: "E", Decimals: 2},
	{Code: "THB", Numeric: "764", Name: "Thai Baht", Symbol: "฿", Decimals: 2},
	{Code: "TJS", Numeric: "972", Name: "Tajikistani Somoni", Symbol: "SM", Decimals: 2},
	{Code: "TMT", Numeric: "934", Name: "Turkmenistani Manat", Symbol: "T", Decimals: 2},
	{Code: "TND", Numeric: "788", Name: "Tunisian Dinar", Symbol: "د.ت", Decimals: 3},
	{Code: "TOP", Numeric: "776", Name: "Tongan Paʻanga", Symbol: "T$", Decimals: 2},
	{Code: "TRY", Numeric: "949", Name: "Turkish Lira", Symbol: "₺", Decimals: 2},
	{Code: "TTD", Numeric: "780", Name: "Trinidad and Tobago Dollar", Symbol: "TT$", Decimals: 2},
	{Code: "TWD", Numeric: "901", Name: "New Taiwan Dollar", Symbol: "NT$", Decimals: 2},
	{Code: "TZS", Numeric: "834", Name: "Tanzanian Shilling", Symbol: "TSh", Decimals: 2},
	{Code: "UAH", Numeric: "980", Name: "Ukrainian Hryvnia", Symbol: "₴", Decimals: 2},
	{Code: "UGX", Numeric: "800", Name: "Ugandan Shilling", Symbol: "USh", Decimals: 0},
	{Code: "USD", Numeric: "840", Name: "US Dollar", Symbol: "$", Decimals: 2},
	{Code: "UYU", Numeric: "858", Name: "Uruguayan Peso", Symbol: "$U", Decimals: 2},
	{Code: "UZS", Numeric: "860", Name: "Uzbekistan Som", Symbol: "сўм", Decimals: 2},
	{Code: "VES", Numeric: "928", Name: "Venezuelan Bolívar", Symbol: "Bs.S", Decimals: 2},
	{Code: "VND", Numeric: "704", Name: "Vietnamese Dong", Symbol: "₫", Decimals: 0},
	{Code: "VUV", Numeric: "548", Name: "Vanuatu Vatu", Symbol: "VT", Decimals: 0},
	{Code: "WST", Numeric: "882", Name: "Samoan Tala", Symbol: "WS$", Decimals: 2},
	{Code: "XAF", Numeric: "950", Name: "CFA Franc BEAC", Symbol: "FCFA", Decimals: 0},
	{Code: "XCD", Numeric: "951", Name: "East Caribbean Dollar", Symbol: "EC$", Decimals: 2},
	{Code: "XOF", Numeric: "952", Name: "CFA Franc BCEAO", Symbol: "CFA", Decimals: 0},
	{Code: "XPF", Numeric: "953", Name: "CFP Franc", Symbol: "₣", Decimals: 0},
	{Code: "YER", Numeric: "886", Name: "Yemeni Rial", Symbol: "﷼", Decimals: 2},
	{Code: "ZAR", Numeric: "710", Name: "South African Rand", Symbol: "R", Decimals: 2},
	{Code: "ZMW", Numeric: "967", Name: "Zambian Kwacha", Symbol: "ZK", Decimals: 2},
	{Code: "ZWL", Numeric: "932", Name: "Zimbabwean Dollar", Symbol: "Z$", Decimals: 2},
}
