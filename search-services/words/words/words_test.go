package words_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/words/words"
)

// Пустая строка возвращает nil
func TestNorm_Empty(t *testing.T) {
	require.Nil(t, words.Norm(""))
}

// Стоп-слова и короткие слова фильтруются полностью
func TestNorm_StopShortWords(t *testing.T) {
	result := words.Norm("a an to be is")
	require.Empty(t, result)
}

// Одинаковые стемы схлопываются в одно слово
func TestNorm_Deduplication(t *testing.T) {
	result := words.Norm("running runs run")
	require.Len(t, result, 1)
}

// Знаки препинания убираются, слова остаются
func TestNorm_Punctuation(t *testing.T) {
	result := words.Norm("hello, world!")
	require.ElementsMatch(t, []string{"hello", "world"}, result)
}

// Разные формы одного слова дают один стем
func TestNorm_Stemming(t *testing.T) {
	result := words.Norm("computers computing")
	require.Len(t, result, 1)
}

// Обычная фраза нормализуется, стоп-слово "the" отбрасывается
func TestNorm_BasicPhrase(t *testing.T) {
	result := words.Norm("the quick brown fox")
	require.NotEmpty(t, result)
	require.Contains(t, result, "quick")
	require.Contains(t, result, "brown")
}

// Регистр не влияет на результат
func TestNorm_MixedCase(t *testing.T) {
	lower := words.Norm("Linux")
	upper := words.Norm("linux")
	require.Equal(t, lower, upper)
}
